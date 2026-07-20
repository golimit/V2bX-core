package conf

import "fmt"

type LimitConfig struct {
	EnableRealtime          bool                     `json:"EnableRealtime"`
	SpeedLimit              int                      `json:"SpeedLimit"`
	IPLimit                 int                      `json:"DeviceLimit"`
	ConnLimit               int                      `json:"ConnLimit"`
	EnableIpRecorder        bool                     `json:"EnableIpRecorder"`
	IpRecorderConfig        *IpReportConfig          `json:"IpRecorderConfig"`
	EnableDynamicSpeedLimit bool                     `json:"EnableDynamicSpeedLimit"`
	DynamicSpeedLimitConfig *DynamicSpeedLimitConfig `json:"DynamicSpeedLimitConfig"`
}

// ValidateDynamicSpeedLimit returns an error when dynamic speed limit is
// enabled but misconfigured (avoids nil panic at runtime).
func (l *LimitConfig) ValidateDynamicSpeedLimit() error {
	if !l.EnableDynamicSpeedLimit {
		return nil
	}
	c := l.DynamicSpeedLimitConfig
	if c == nil {
		return fmt.Errorf("EnableDynamicSpeedLimit is true but DynamicSpeedLimitConfig is nil")
	}
	if c.Periodic <= 0 {
		return fmt.Errorf("DynamicSpeedLimitConfig.Periodic must be > 0 (seconds)")
	}
	if c.Traffic <= 0 {
		return fmt.Errorf("DynamicSpeedLimitConfig.Traffic must be > 0 (bytes)")
	}
	if c.SpeedLimit <= 0 {
		return fmt.Errorf("DynamicSpeedLimitConfig.SpeedLimit must be > 0 (Mbps)")
	}
	if c.ExpireTime <= 0 {
		return fmt.Errorf("DynamicSpeedLimitConfig.ExpireTime must be > 0 (minutes)")
	}
	return nil
}

type RecorderConfig struct {
	Url     string `json:"Url"`
	Token   string `json:"Token"`
	Timeout int    `json:"Timeout"`
}

type RedisConfig struct {
	Address  string `json:"Address"`
	Password string `json:"Password"`
	Db       int    `json:"Db"`
	Expiry   int    `json:"Expiry"`
}

type IpReportConfig struct {
	Periodic       int             `json:"Periodic"`
	Type           string          `json:"Type"`
	RecorderConfig *RecorderConfig `json:"RecorderConfig"`
	RedisConfig    *RedisConfig    `json:"RedisConfig"`
	EnableIpSync   bool            `json:"EnableIpSync"`
}

type DynamicSpeedLimitConfig struct {
	Periodic   int   `json:"Periodic"`
	Traffic    int64 `json:"Traffic"`
	SpeedLimit int   `json:"SpeedLimit"`
	ExpireTime int   `json:"ExpireTime"`
}
