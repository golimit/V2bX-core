package hy2

import (
	"fmt"
	"net"
	"sync"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/common/counter"
	vCore "github.com/InazumaV/V2bX/core"
	"github.com/apernet/hysteria/core/v2/server"
)

var _ server.Authenticator = &V2bX{}

type V2bX struct {
	usersMap map[string]int
	mutex    sync.RWMutex
}

func newNodeAuth() *V2bX {
	return &V2bX{
		usersMap: make(map[string]int),
	}
}

func (v *V2bX) Authenticate(addr net.Addr, auth string, tx uint64) (ok bool, id string) {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	if _, exists := v.usersMap[auth]; exists {
		return true, auth
	}
	return false, ""
}

func (h *Hysteria2) AddUsers(p *vCore.AddUsersParams) (added int, err error) {
	node, ok := h.Hy2nodes[p.Tag]
	if !ok {
		return 0, fmt.Errorf("node %s not found", p.Tag)
	}
	node.Auth.mutex.Lock()
	defer node.Auth.mutex.Unlock()
	for _, user := range p.Users {
		node.Auth.usersMap[user.Uuid] = user.Id
	}
	return len(p.Users), nil
}

func (h *Hysteria2) DelUsers(users []panel.UserInfo, tag string, _ *panel.NodeInfo) error {
	node, ok := h.Hy2nodes[tag]
	if !ok {
		return fmt.Errorf("node %s not found", tag)
	}
	if v, ok := node.TrafficLogger.(*HookServer).Counter.Load(tag); ok {
		c := v.(*counter.TrafficCounter)
		for _, user := range users {
			c.Delete(user.Uuid)
		}
	}
	node.Auth.mutex.Lock()
	defer node.Auth.mutex.Unlock()
	for _, user := range users {
		delete(node.Auth.usersMap, user.Uuid)
	}
	return nil
}

func (h *Hysteria2) GetUserTrafficSlice(tag string, reset bool) ([]panel.UserTraffic, error) {
	trafficSlice := make([]panel.UserTraffic, 0)
	node, ok := h.Hy2nodes[tag]
	if !ok {
		return nil, nil
	}
	node.Auth.mutex.RLock()
	defer node.Auth.mutex.RUnlock()
	hook := node.TrafficLogger.(*HookServer)
	if v, ok := hook.Counter.Load(tag); ok {
		c := v.(*counter.TrafficCounter)
		c.Counters.Range(func(key, value interface{}) bool {
			uuid := key.(string)
			traffic := value.(*counter.TrafficStorage)
			var up, down int64
			if reset {
				up = traffic.UpCounter.Swap(0)
				down = traffic.DownCounter.Swap(0)
			} else {
				up = traffic.UpCounter.Load()
				down = traffic.DownCounter.Load()
			}
			if up+down > hook.ReportMinTrafficBytes {
				if node.Auth.usersMap[uuid] == 0 {
					c.Delete(uuid)
					return true
				}
				trafficSlice = append(trafficSlice, panel.UserTraffic{
					UID:      node.Auth.usersMap[uuid],
					Upload:   up,
					Download: down,
				})
			} else if reset && (up != 0 || down != 0) {
				traffic.UpCounter.Add(up)
				traffic.DownCounter.Add(down)
			}
			return true
		})
		if len(trafficSlice) == 0 {
			return nil, nil
		}
		return trafficSlice, nil
	}
	return nil, nil
}

func (h *Hysteria2) AddUserTraffic(tag string, report []panel.UserTraffic) {
	if len(report) == 0 {
		return
	}
	node, ok := h.Hy2nodes[tag]
	if !ok {
		return
	}
	hook := node.TrafficLogger.(*HookServer)
	v, ok := hook.Counter.Load(tag)
	if !ok {
		return
	}
	c := v.(*counter.TrafficCounter)
	node.Auth.mutex.RLock()
	defer node.Auth.mutex.RUnlock()
	uidToUUID := make(map[int]string, len(node.Auth.usersMap))
	for uuid, uid := range node.Auth.usersMap {
		uidToUUID[uid] = uuid
	}
	for i := range report {
		uuid, ok := uidToUUID[report[i].UID]
		if !ok {
			continue
		}
		c.Add(uuid, report[i].Upload, report[i].Download)
	}
}
