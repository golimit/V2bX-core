package limiter

// determineSpeedLimit returns the minimum non-zero rate (Mbps).
// Zero means "unlimited" on that side and is ignored.
func determineSpeedLimit(limit1, limit2 int) int {
	if limit1 == 0 {
		return limit2
	}
	if limit2 == 0 {
		return limit1
	}
	if limit1 < limit2 {
		return limit1
	}
	return limit2
}

func (u *UserLimitInfo) effectiveUserLimit(now int64) (speedLimit int, cleared bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ExpireTime != 0 && u.ExpireTime < now {
		u.DynamicSpeedLimit = 0
		u.ExpireTime = 0
		cleared = true
	}
	return determineSpeedLimit(u.SpeedLimit, u.DynamicSpeedLimit), cleared
}

func (u *UserLimitInfo) snapshotDevice() (uid, deviceLimit int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.UID, u.DeviceLimit
}
