package node

import (
	"strconv"

	"github.com/InazumaV/V2bX/api/panel"
	log "github.com/sirupsen/logrus"
)

func (c *Controller) reportUserTrafficTask() (err error) {
	userTraffic, err := c.server.GetUserTrafficSlice(c.tag, true)
	if err != nil {
		log.WithFields(log.Fields{
			"tag": c.tag,
			"err": err,
		}).Error("Get user traffic failed")
		return nil
	}
	if len(userTraffic) > 0 {
		if err = c.apiClient.ReportUserTraffic(userTraffic); err != nil {
			c.server.AddUserTraffic(c.tag, userTraffic)
			log.WithFields(log.Fields{
				"tag": c.tag,
				"err": err,
			}).Info("Report user traffic failed, restored counters")
		} else {
			if c.LimitConfig.EnableDynamicSpeedLimit {
				c.accumulateDynamicTraffic(userTraffic)
			}
			log.WithField("tag", c.tag).Infof("Report %d users traffic", len(userTraffic))
			log.WithField("tag", c.tag).Debugf("User traffic: %+v", userTraffic)
		}
	}

	if onlineDevice, err := c.limiter.GetOnlineDevice(); err != nil {
		log.WithFields(log.Fields{
			"tag": c.tag,
			"err": err,
		}).Error("Get online device failed")
	} else if len(*onlineDevice) > 0 {
		var result []panel.OnlineUser
		var nocountUID = make(map[int]struct{})
		for _, traffic := range userTraffic {
			total := traffic.Upload + traffic.Download
			if total < int64(c.Options.DeviceOnlineMinTraffic*1000) {
				nocountUID[traffic.UID] = struct{}{}
			}
		}
		for _, online := range *onlineDevice {
			if _, ok := nocountUID[online.UID]; !ok {
				result = append(result, online)
			}
		}
		data := make(map[int][]string)
		for _, onlineuser := range result {
			data[onlineuser.UID] = append(data[onlineuser.UID], onlineuser.IP)
		}
		if err = c.apiClient.ReportNodeOnlineUsers(&data); err != nil {
			log.WithFields(log.Fields{
				"tag": c.tag,
				"err": err,
			}).Info("Report online users failed")
		} else {
			log.WithField("tag", c.tag).Infof("Total %d online users, %d Reported", len(*onlineDevice), len(result))
			log.WithField("tag", c.tag).Debugf("Online users: %+v", data)
		}
	}

	return nil
}

// flushUserTraffic reports any pending traffic before user/node teardown.
// On failure, counters are restored and a non-nil error is returned so callers
// must abort DelNode/DelUsers to avoid dropping restored traffic.
func (c *Controller) flushUserTraffic() error {
	userTraffic, err := c.server.GetUserTrafficSlice(c.tag, true)
	if err != nil {
		log.WithFields(log.Fields{
			"tag": c.tag,
			"err": err,
		}).Error("Flush get user traffic failed")
		return err
	}
	if len(userTraffic) == 0 {
		return nil
	}
	if err = c.apiClient.ReportUserTraffic(userTraffic); err != nil {
		c.server.AddUserTraffic(c.tag, userTraffic)
		log.WithFields(log.Fields{
			"tag": c.tag,
			"err": err,
		}).Error("Flush report user traffic failed, restored counters")
		return err
	}
	if c.LimitConfig.EnableDynamicSpeedLimit {
		c.accumulateDynamicTraffic(userTraffic)
	}
	log.WithField("tag", c.tag).Infof("Flushed %d users traffic", len(userTraffic))
	return nil
}

func (c *Controller) accumulateDynamicTraffic(userTraffic []panel.UserTraffic) {
	if c.traffic == nil {
		return
	}
	uidToUUID := make(map[int]string, len(c.userList))
	for i := range c.userList {
		uidToUUID[c.userList[i].Id] = c.userList[i].Uuid
	}
	for i := range userTraffic {
		uuid, ok := uidToUUID[userTraffic[i].UID]
		if !ok {
			continue
		}
		c.traffic.add(uuid, userTraffic[i].Upload+userTraffic[i].Download)
	}
}

func compareUserList(old, new []panel.UserInfo) (deleted, added []panel.UserInfo) {
	oldMap := make(map[string]int)
	for i, user := range old {
		key := user.Uuid + strconv.Itoa(user.SpeedLimit) + "|" + strconv.Itoa(user.DeviceLimit)
		oldMap[key] = i
	}

	for _, user := range new {
		key := user.Uuid + strconv.Itoa(user.SpeedLimit) + "|" + strconv.Itoa(user.DeviceLimit)
		if _, exists := oldMap[key]; !exists {
			added = append(added, user)
		} else {
			delete(oldMap, key)
		}
	}

	for _, index := range oldMap {
		deleted = append(deleted, old[index])
	}

	return deleted, added
}
