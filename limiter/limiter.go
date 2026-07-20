package limiter

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/common/format"
	"github.com/InazumaV/V2bX/conf"
	"github.com/juju/ratelimit"
)

var limitLock sync.RWMutex
var limiter map[string]*Limiter

func Init() {
	limiter = map[string]*Limiter{}
}

// speedBucket pairs a token bucket with its configured rate so CheckLimit
// can rebuild when the effective limit changes.
type speedBucket struct {
	bucket  *ratelimit.Bucket
	rateBps int64
}

type Limiter struct {
	DomainRules   []*regexp.Regexp
	ProtocolRules []string
	SpeedLimit    int
	UserOnlineIP  *sync.Map // Key: TagUUID, value: {Key: Ip, value: Uid}
	OldUserOnline *sync.Map // Key: Ip, value: Uid
	UserLimitInfo *sync.Map // Key: TagUUID value: *UserLimitInfo
	SpeedLimiter  *sync.Map // key: TagUUID, value: *speedBucket

	mu        sync.RWMutex
	UUIDtoUID map[string]int // Key: UUID, value: Uid
	AliveList map[int]int    // Key: Uid, value: alive_ip
}

type UserLimitInfo struct {
	mu                sync.Mutex
	UID               int
	SpeedLimit        int
	DeviceLimit       int
	DynamicSpeedLimit int
	ExpireTime        int64
	OverLimit         bool
}

func AddLimiter(tag string, l *conf.LimitConfig, users []panel.UserInfo, aliveList map[int]int) *Limiter {
	if aliveList == nil {
		aliveList = make(map[int]int)
	}
	info := &Limiter{
		SpeedLimit:    l.SpeedLimit,
		UserOnlineIP:  new(sync.Map),
		UserLimitInfo: new(sync.Map),
		SpeedLimiter:  new(sync.Map),
		AliveList:     aliveList,
		OldUserOnline: new(sync.Map),
	}
	uuidmap := make(map[string]int, len(users))
	for i := range users {
		uuidmap[users[i].Uuid] = users[i].Id
		userLimit := &UserLimitInfo{
			UID: users[i].Id,
		}
		if users[i].SpeedLimit != 0 {
			userLimit.SpeedLimit = users[i].SpeedLimit
		}
		if users[i].DeviceLimit != 0 {
			userLimit.DeviceLimit = users[i].DeviceLimit
		}
		info.UserLimitInfo.Store(format.UserTag(tag, users[i].Uuid), userLimit)
	}
	info.UUIDtoUID = uuidmap
	limitLock.Lock()
	limiter[tag] = info
	limitLock.Unlock()
	return info
}

func GetLimiter(tag string) (info *Limiter, err error) {
	limitLock.RLock()
	info, ok := limiter[tag]
	limitLock.RUnlock()
	if !ok {
		return nil, errors.New("not found")
	}
	return info, nil
}

func DeleteLimiter(tag string) {
	limitLock.Lock()
	delete(limiter, tag)
	limitLock.Unlock()
}

func (l *Limiter) UpdateUser(tag string, added []panel.UserInfo, deleted []panel.UserInfo) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range deleted {
		key := format.UserTag(tag, deleted[i].Uuid)
		l.UserLimitInfo.Delete(key)
		l.UserOnlineIP.Delete(key)
		l.SpeedLimiter.Delete(key)
		delete(l.UUIDtoUID, deleted[i].Uuid)
		delete(l.AliveList, deleted[i].Id)
	}
	for i := range added {
		userLimit := &UserLimitInfo{
			UID: added[i].Id,
		}
		if added[i].SpeedLimit != 0 {
			userLimit.SpeedLimit = added[i].SpeedLimit
		}
		if added[i].DeviceLimit != 0 {
			userLimit.DeviceLimit = added[i].DeviceLimit
		}
		l.UserLimitInfo.Store(format.UserTag(tag, added[i].Uuid), userLimit)
		l.UUIDtoUID[added[i].Uuid] = added[i].Id
	}
}

func (l *Limiter) SetAliveList(alive map[int]int) {
	if alive == nil {
		return
	}
	l.mu.Lock()
	l.AliveList = alive
	l.mu.Unlock()
}

func (l *Limiter) CloneAliveList() map[int]int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.AliveList == nil {
		return make(map[int]int)
	}
	out := make(map[int]int, len(l.AliveList))
	for k, v := range l.AliveList {
		out[k] = v
	}
	return out
}

// UpdateDynamicSpeedLimit sets a temporary speed limit and invalidates any
// cached token bucket so the next CheckLimit rebuilds at the new rate.
func (l *Limiter) UpdateDynamicSpeedLimit(tag, uuid string, limit int, expire time.Time) error {
	key := format.UserTag(tag, uuid)
	v, ok := l.UserLimitInfo.Load(key)
	if !ok {
		return errors.New("not found")
	}
	info := v.(*UserLimitInfo)
	info.mu.Lock()
	info.DynamicSpeedLimit = limit
	info.ExpireTime = expire.Unix()
	info.mu.Unlock()
	l.SpeedLimiter.Delete(key)
	return nil
}

func (l *Limiter) invalidateSpeedBucket(taguuid string) {
	l.SpeedLimiter.Delete(taguuid)
}

func (l *Limiter) CheckLimit(taguuid string, ip string, isTcp bool, noSSUDP bool) (Bucket *ratelimit.Bucket, Reject bool) {
	ip = strings.TrimPrefix(ip, "::ffff:")

	nodeLimit := l.SpeedLimit
	userLimit := 0
	deviceLimit := 0
	var uid int
	v, ok := l.UserLimitInfo.Load(taguuid)
	if !ok {
		return nil, true
	}
	u := v.(*UserLimitInfo)
	uid, deviceLimit = u.snapshotDevice()
	var cleared bool
	userLimit, cleared = u.effectiveUserLimit(time.Now().Unix())
	if cleared {
		l.invalidateSpeedBucket(taguuid)
	}

	if noSSUDP {
		l.mu.RLock()
		aliveIp := l.AliveList[uid]
		l.mu.RUnlock()

		if v, ok := l.UserOnlineIP.Load(taguuid); ok {
			ipMap := v.(*sync.Map)
			if _, loaded := ipMap.LoadOrStore(ip, uid); !loaded {
				if ov, loaded := l.OldUserOnline.Load(ip); loaded {
					if ov.(int) == uid {
						l.OldUserOnline.Delete(ip)
					}
				} else if deviceLimit > 0 && deviceLimit <= aliveIp {
					ipMap.Delete(ip)
					return nil, true
				}
			}
		} else {
			ipMap := new(sync.Map)
			ipMap.Store(ip, uid)
			if actual, loaded := l.UserOnlineIP.LoadOrStore(taguuid, ipMap); loaded {
				existing := actual.(*sync.Map)
				if _, ipLoaded := existing.LoadOrStore(ip, uid); !ipLoaded {
					if ov, ok := l.OldUserOnline.Load(ip); ok {
						if ov.(int) == uid {
							l.OldUserOnline.Delete(ip)
						}
					} else if deviceLimit > 0 && deviceLimit <= aliveIp {
						existing.Delete(ip)
						return nil, true
					}
				}
			} else if ov, ok := l.OldUserOnline.Load(ip); ok {
				if ov.(int) == uid {
					l.OldUserOnline.Delete(ip)
				}
			} else if deviceLimit > 0 && deviceLimit <= aliveIp {
				l.UserOnlineIP.Delete(taguuid)
				return nil, true
			}
		}
	}

	limit := int64(determineSpeedLimit(nodeLimit, userLimit)) * 1000000 / 8
	if limit <= 0 {
		l.SpeedLimiter.Delete(taguuid)
		return nil, false
	}

	if v, ok := l.SpeedLimiter.Load(taguuid); ok {
		sb := v.(*speedBucket)
		if sb.rateBps == limit {
			return sb.bucket, false
		}
		l.SpeedLimiter.Delete(taguuid)
	}
	bucket := ratelimit.NewBucketWithQuantum(time.Second, limit, limit)
	sb := &speedBucket{bucket: bucket, rateBps: limit}
	if actual, loaded := l.SpeedLimiter.LoadOrStore(taguuid, sb); loaded {
		existing := actual.(*speedBucket)
		if existing.rateBps == limit {
			return existing.bucket, false
		}
		// Another goroutine stored a different rate; prefer ours.
		l.SpeedLimiter.Store(taguuid, sb)
	}
	return bucket, false
}

func (l *Limiter) GetOnlineDevice() (*[]panel.OnlineUser, error) {
	var onlineUser []panel.OnlineUser
	newOld := new(sync.Map)
	l.UserOnlineIP.Range(func(key, value interface{}) bool {
		taguuid := key.(string)
		ipMap := value.(*sync.Map)
		ipMap.Range(func(key, value interface{}) bool {
			uid := value.(int)
			ip := key.(string)
			newOld.Store(ip, uid)
			onlineUser = append(onlineUser, panel.OnlineUser{UID: uid, IP: ip})
			return true
		})
		l.UserOnlineIP.Delete(taguuid)
		return true
	})
	l.OldUserOnline = newOld
	return &onlineUser, nil
}

type UserIpList struct {
	Uid    int      `json:"Uid"`
	IpList []string `json:"Ips"`
}
