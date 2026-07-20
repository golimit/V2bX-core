package node

import (
	"os"
	"time"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/common/task"
	vCore "github.com/InazumaV/V2bX/core"
	"github.com/InazumaV/V2bX/limiter"
	log "github.com/sirupsen/logrus"
)

func (c *Controller) startTasks(node *panel.NodeInfo) {
	// fetch node info task
	c.nodeInfoMonitorPeriodic = &task.Task{
		Interval: node.PullInterval,
		Execute:  c.nodeInfoMonitor,
	}
	// fetch user list task
	c.userReportPeriodic = &task.Task{
		Interval: node.PushInterval,
		Execute:  c.reportUserTrafficTask,
	}
	log.WithField("tag", c.tag).Info("Start monitor node status")
	_ = c.nodeInfoMonitorPeriodic.Start(false)
	log.WithField("tag", c.tag).Info("Start report node status")
	_ = c.userReportPeriodic.Start(false)
	if node.Security == panel.Tls {
		switch c.CertConfig.CertMode {
		case "none", "", "file", "self":
		default:
			c.renewCertPeriodic = &task.Task{
				Interval: time.Hour * 24,
				Execute:  c.renewCertTask,
			}
			log.WithField("tag", c.tag).Info("Start renew cert")
			_ = c.renewCertPeriodic.Start(true)
		}
	}
	if c.LimitConfig.EnableDynamicSpeedLimit {
		if err := c.LimitConfig.ValidateDynamicSpeedLimit(); err != nil {
			log.WithFields(log.Fields{
				"tag": c.tag,
				"err": err,
			}).Error("Skip dynamic speed limit: invalid config")
		} else {
			c.traffic = newDynamicTraffic()
			c.dynamicSpeedLimitPeriodic = &task.Task{
				Interval: time.Duration(c.LimitConfig.DynamicSpeedLimitConfig.Periodic) * time.Second,
				Execute:  c.SpeedChecker,
			}
			log.WithField("tag", c.tag).Infof(
				"Start dynamic speed limit (period=%ds traffic=%dB limit=%dMbps expire=%dm)",
				c.LimitConfig.DynamicSpeedLimitConfig.Periodic,
				c.LimitConfig.DynamicSpeedLimitConfig.Traffic,
				c.LimitConfig.DynamicSpeedLimitConfig.SpeedLimit,
				c.LimitConfig.DynamicSpeedLimitConfig.ExpireTime,
			)
			_ = c.dynamicSpeedLimitPeriodic.Start(false)
		}
	}
}

func (c *Controller) nodeInfoMonitor() (err error) {
	newN, err := c.apiClient.GetNodeInfo()
	if err != nil {
		log.WithFields(log.Fields{
			"tag": c.tag,
			"err": err,
		}).Error("Get node info failed")
		return nil
	}
	newU, err := c.apiClient.GetUserList()
	if err != nil {
		log.WithFields(log.Fields{
			"tag": c.tag,
			"err": err,
		}).Error("Get user list failed")
		return nil
	}
	newA, err := c.apiClient.GetUserAlive()
	if err != nil {
		log.WithFields(log.Fields{
			"tag": c.tag,
			"err": err,
		}).Error("Get alive list failed, keep previous")
		newA = nil
	}
	if newN != nil {
		oldUserList := c.userList
		oldInfo := c.info
		c.info = newN
		var removed []panel.UserInfo
		if newU != nil {
			removed, _ = compareUserList(oldUserList, newU)
			c.userList = newU
		}
		log.WithField("tag", c.tag).Info("Node changed, reload")
		if err = c.flushUserTraffic(); err != nil {
			log.WithFields(log.Fields{
				"tag": c.tag,
				"err": err,
			}).Error("Abort node reload: flush traffic failed")
			return nil
		}
		if c.traffic != nil {
			c.traffic.reset()
		}
		if len(removed) > 0 && oldInfo != nil {
			if err = c.server.DelUsers(removed, c.tag, oldInfo); err != nil {
				log.WithFields(log.Fields{
					"tag": c.tag,
					"err": err,
				}).Error("Delete users before node reload failed")
				return nil
			}
			c.limiter.UpdateUser(c.tag, nil, removed)
			if c.traffic != nil {
				for i := range removed {
					c.traffic.remove(removed[i].Uuid)
				}
			}
		}
		err = c.server.DelNode(c.tag)
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.tag,
				"err": err,
			}).Error("Delete node failed")
			return nil
		}

		if len(c.Options.Name) == 0 {
			oldTag := c.tag
			c.tag = c.buildNodeTag(newN)
			limiter.DeleteLimiter(oldTag)
			alive := newA
			if alive == nil && c.limiter != nil {
				alive = c.limiter.CloneAliveList()
			}
			l := limiter.AddLimiter(c.tag, &c.LimitConfig, c.userList, alive)
			c.limiter = l
		} else if newA != nil {
			c.limiter.SetAliveList(newA)
		}
		err = c.limiter.UpdateRule(&newN.Rules)
		if err != nil {
			c.fatalHalfState("update rule after DelNode", err)
			return nil
		}

		if newN.Security == panel.Tls {
			err = c.requestCert()
			if err != nil {
				c.fatalHalfState("request cert after DelNode", err)
				return nil
			}
		}
		err = c.server.AddNode(c.tag, newN, c.Options)
		if err != nil {
			c.fatalHalfState("AddNode after DelNode", err)
			return nil
		}
		_, err = c.server.AddUsers(&vCore.AddUsersParams{
			Tag:      c.tag,
			Users:    c.userList,
			NodeInfo: newN,
		})
		if err != nil {
			c.fatalHalfState("AddUsers after DelNode", err)
			return nil
		}
		if c.nodeInfoMonitorPeriodic.Interval != newN.PullInterval &&
			newN.PullInterval != 0 {
			c.nodeInfoMonitorPeriodic.Interval = newN.PullInterval
			c.nodeInfoMonitorPeriodic.Close()
			_ = c.nodeInfoMonitorPeriodic.Start(false)
		}
		if c.userReportPeriodic.Interval != newN.PushInterval &&
			newN.PushInterval != 0 {
			c.userReportPeriodic.Interval = newN.PushInterval
			c.userReportPeriodic.Close()
			_ = c.userReportPeriodic.Start(false)
		}
		log.WithField("tag", c.tag).Infof("Added %d new users", len(c.userList))
		return nil
	}
	if newA != nil {
		c.limiter.SetAliveList(newA)
	}
	if len(newU) == 0 {
		return nil
	}
	deleted, added := compareUserList(c.userList, newU)
	if len(deleted) > 0 {
		if err = c.flushUserTraffic(); err != nil {
			log.WithFields(log.Fields{
				"tag": c.tag,
				"err": err,
			}).Error("Abort DelUsers: flush traffic failed")
			return nil
		}
		err = c.server.DelUsers(deleted, c.tag, c.info)
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.tag,
				"err": err,
			}).Error("Delete users failed")
			return nil
		}
	}
	if len(added) > 0 {
		_, err = c.server.AddUsers(&vCore.AddUsersParams{
			Tag:      c.tag,
			NodeInfo: c.info,
			Users:    added,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.tag,
				"err": err,
			}).Error("Add users failed")
			return nil
		}
	}
	if len(added) > 0 || len(deleted) > 0 {
		c.limiter.UpdateUser(c.tag, added, deleted)
		if c.traffic != nil {
			for i := range deleted {
				c.traffic.remove(deleted[i].Uuid)
			}
		}
	}
	c.userList = newU
	if len(added)+len(deleted) != 0 {
		log.WithField("tag", c.tag).
			Infof("%d user deleted, %d user added", len(deleted), len(added))
	}
	return nil
}

// fatalHalfState exits after DelNode succeeded but recovery failed,
// so Docker restart:always can restore service.
func (c *Controller) fatalHalfState(op string, err error) {
	log.WithFields(log.Fields{
		"tag": c.tag,
		"op":  op,
		"err": err,
	}).Error("Node half-state after DelNode; exiting for restart")
	os.Exit(1)
}

func (c *Controller) SpeedChecker() error {
	cfg := c.LimitConfig.DynamicSpeedLimitConfig
	if cfg == nil || c.traffic == nil || c.limiter == nil {
		return nil
	}
	snap := c.traffic.snapshot()
	expireAt := time.Now().Add(time.Duration(cfg.ExpireTime) * time.Minute)
	for uuid, bytes := range snap {
		if bytes < cfg.Traffic {
			continue
		}
		if err := c.limiter.UpdateDynamicSpeedLimit(c.tag, uuid, cfg.SpeedLimit, expireAt); err != nil {
			log.WithFields(log.Fields{
				"tag":  c.tag,
				"uuid": uuid,
				"err":  err,
			}).Error("Update dynamic speed limit failed")
			continue
		}
		c.traffic.remove(uuid)
		log.WithFields(log.Fields{
			"tag":   c.tag,
			"uuid":  uuid,
			"bytes": bytes,
			"limit": cfg.SpeedLimit,
		}).Info("Dynamic speed limit applied")
	}
	return nil
}
