package conf

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"encoding/json"

	"github.com/InazumaV/V2bX/common/json5"
)

type NodeConfig struct {
	ApiConfig ApiConfig `json:"-"`
	Options   Options   `json:"-"`
}

type rawNodeConfig struct {
	Include string          `json:"Include"`
	ApiRaw  json.RawMessage `json:"ApiConfig"`
	OptRaw  json.RawMessage `json:"Options"`
}

type ApiConfig struct {
	APIHost      string `json:"ApiHost"`
	APISendIP    string `json:"ApiSendIP"`
	NodeID       int    `json:"NodeID"`
	Key          string `json:"ApiKey"`
	NodeType     string `json:"NodeType"`
	Timeout      int    `json:"Timeout"`
	RuleListPath string `json:"RuleListPath"`
}

func (n *NodeConfig) UnmarshalJSON(data []byte) (err error) {
	rn := rawNodeConfig{}
	err = json.Unmarshal(data, &rn)
	if err != nil {
		return err
	}
	if len(rn.Include) != 0 {
		include := rn.Include
		switch {
		case strings.HasPrefix(include, "http://") || strings.HasPrefix(include, "https://"):
			rsp, err := http.Get(include)
			if err != nil {
				return err
			}
			defer rsp.Body.Close()
			data, err = io.ReadAll(json5.NewTrimNodeReader(rsp.Body))
			if err != nil {
				return fmt.Errorf("open include file error: %s", err)
			}
		default:
			// trim optional "file:" prefix
			path := strings.TrimPrefix(include, "file:")
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("open include file error: %s", err)
			}
			defer f.Close()
			data, err = io.ReadAll(json5.NewTrimNodeReader(f))
			if err != nil {
				return fmt.Errorf("open include file error: %s", err)
			}
		}
		err = json.Unmarshal(data, &rn)
		if err != nil {
			return fmt.Errorf("unmarshal include file error: %s", err)
		}
	}

	n.ApiConfig = ApiConfig{
		APIHost: "http://127.0.0.1",
		Timeout: 30,
	}
	if len(rn.ApiRaw) > 0 {
		err = json.Unmarshal(rn.ApiRaw, &n.ApiConfig)
		if err != nil {
			return
		}
	} else {
		err = json.Unmarshal(data, &n.ApiConfig)
		if err != nil {
			return
		}
	}

	n.Options = Options{
		ListenIP:   "0.0.0.0",
		SendIP:     "0.0.0.0",
		CertConfig: NewCertConfig(),
	}
	if len(rn.OptRaw) > 0 {
		err = json.Unmarshal(rn.OptRaw, &n.Options)
		if err != nil {
			return
		}
	} else {
		err = json.Unmarshal(data, &n.Options)
		if err != nil {
			return
		}
	}
	return
}

type Options struct {
	Name                   string          `json:"Name"`
	Core                   string          `json:"Core"`
	CoreName               string          `json:"CoreName"`
	ListenIP               string          `json:"ListenIP"`
	SendIP                 string          `json:"SendIP"`
	DeviceOnlineMinTraffic int64           `json:"DeviceOnlineMinTraffic"`
	ReportMinTraffic       int64           `json:"ReportMinTraffic"`
	LimitConfig            LimitConfig     `json:"LimitConfig"`
	RawOptions             json.RawMessage `json:"RawOptions"`
	XrayOptions            *XrayOptions    `json:"XrayOptions"`
	SingOptions            *SingOptions    `json:"SingOptions"`
	Hysteria2ConfigPath    string          `json:"Hysteria2ConfigPath"`
	CertConfig             *CertConfig     `json:"CertConfig"`
}

func (o *Options) UnmarshalJSON(data []byte) error {
	type opt Options
	err := json.Unmarshal(data, (*opt)(o))
	if err != nil {
		return err
	}
	// Compat aliases for older sample configs
	var alias struct {
		MinReportTraffic int64 `json:"MinReportTraffic"`
	}
	_ = json.Unmarshal(data, &alias)
	if o.ReportMinTraffic == 0 && alias.MinReportTraffic != 0 {
		o.ReportMinTraffic = alias.MinReportTraffic
	}
	switch o.Core {
	case "xray":
		o.XrayOptions = NewXrayOptions()
		return json.Unmarshal(data, o.XrayOptions)
	case "sing":
		o.SingOptions = NewSingOptions()
		if err := json.Unmarshal(data, o.SingOptions); err != nil {
			return err
		}
		var singAlias struct {
			TCPFastOpen  *bool `json:"TCPFastOpen"`
			SniffEnabled *bool `json:"SniffEnabled"`
		}
		_ = json.Unmarshal(data, &singAlias)
		// Only apply alias when canonical keys were absent (zero-value ambiguity:
		// if EnableTFO was explicitly false we cannot tell; alias fills when
		// canonical decode left defaults and alias is set).
		if singAlias.TCPFastOpen != nil {
			// Prefer explicit legacy key when present alongside missing EnableTFO in raw
			var raw map[string]json.RawMessage
			if json.Unmarshal(data, &raw) == nil {
				if _, has := raw["EnableTFO"]; !has {
					o.SingOptions.TCPFastOpen = *singAlias.TCPFastOpen
				}
				if _, has := raw["EnableSniff"]; !has && singAlias.SniffEnabled != nil {
					o.SingOptions.SniffEnabled = *singAlias.SniffEnabled
				}
			}
		} else if singAlias.SniffEnabled != nil {
			var raw map[string]json.RawMessage
			if json.Unmarshal(data, &raw) == nil {
				if _, has := raw["EnableSniff"]; !has {
					o.SingOptions.SniffEnabled = *singAlias.SniffEnabled
				}
			}
		}
		return nil
	case "hysteria2":
		o.RawOptions = data
		return nil
	default:
		o.Core = ""
		o.RawOptions = data
	}
	return nil
}
