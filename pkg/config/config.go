// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/spidernet-io/egressgateway/pkg/iptables"
	"github.com/spidernet-io/egressgateway/pkg/logger"
)

type Config struct {
	// From environment
	EnvConfig

	// FileConfig from configmap
	FileConfig FileConfig

	KubeConfig *rest.Config `json:"-"`
}

func (cfg *Config) PrintPrettyConfig() {
	raw, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(raw))
}

type EnvConfig struct {
	NodeName                  string        `mapstructure:"NODE_NAME"`
	LeaderElection            bool          `mapstructure:"LEADER_ELECTION"`
	LeaderElectionNamespace   string        `mapstructure:"LEADER_ELECTION_NAMESPACE"`
	LeaderElectionID          string        `mapstructure:"LEADER_ELECTION_ID"`
	LeaderElectionLostRestart bool          `mapstructure:"LEADER_ELECTION_LOST_RESTART"`
	MetricsBindAddress        string        `mapstructure:"METRICS_BIND_ADDRESS"`
	HealthProbeBindAddress    string        `mapstructure:"HEALTH_PROBE_BIND_ADDRESS"`
	GopsPort                  int           `mapstructure:"GOPS_PORT"`
	WebhookPort               int           `mapstructure:"WEBHOOK_PORT"`
	PyroscopeServerAddr       string        `mapstructure:"PYROSCOPE_SERVER_ADDR"`
	PodName                   string        `mapstructure:"POD_NAME"`
	PodNamespace              string        `mapstructure:"POD_NAMESPACE"`
	GolangMaxProcs            int32         `mapstructure:"GOLANG_MAX_PROCS"`
	TLSCertDir                string        `mapstructure:"TLS_CERT_DIR"`
	ConfigMapPath             string        `mapstructure:"CONFIGMAP_PATH"`
	UseDevMode                bool          `mapstructure:"LOG_USE_DEV_MODE"`
	Level                     string        `mapstructure:"LOG_LEVEL"`
	WithCaller                bool          `mapstructure:"LOG_WITH_CALLER"`
	Encoder                   string        `mapstructure:"LOG_ENCODER"`
	Logger                    logger.Config `json:"-"`
}

type FileConfig struct {
	EnableIPv4                   bool           `yaml:"enableIPv4"`
	EnableIPv6                   bool           `yaml:"enableIPv6"`
	IPTables                     IPTables       `yaml:"iptables"`
	DatapathMode                 string         `yaml:"datapathMode"`
	TunnelIpv4Subnet             string         `yaml:"tunnelIpv4Subnet"`
	TunnelIpv6Subnet             string         `yaml:"tunnelIpv6Subnet"`
	TunnelIPv4Net                *net.IPNet     `json:"-"`
	TunnelIPv6Net                *net.IPNet     `json:"-"`
	TunnelDetectMethod           string         `yaml:"tunnelDetectMethod"`
	VXLAN                        VXLAN          `yaml:"vxlan"`
	MaxNumberEndpointPerSlice    int            `yaml:"maxNumberEndpointPerSlice"`
	Mark                         string         `yaml:"mark"`
	AnnouncedInterfacesToExclude []string       `yaml:"announcedInterfacesToExclude"`
	AnnounceExcludeRegexp        *regexp.Regexp `json:"-"`
}

const TunnelInterfaceDefaultRoute = "defaultRouteInterface"
const TunnelInterfaceSpecific = "interface="

type VXLAN struct {
	Name                   string `yaml:"name"`
	ID                     int    `yaml:"id"`
	Port                   int    `yaml:"port"`
	DisableChecksumOffload bool   `yaml:"disableChecksumOffload"`
}

type IPTables struct {
	BackendMode                    string `yaml:"backendMode"`
	RefreshIntervalSecond          int    `yaml:"refreshIntervalSecond"`
	PostWriteIntervalSecond        int    `yaml:"postWriteIntervalSecond"`
	LockTimeoutSecond              int    `yaml:"lockTimeoutSecond"`
	LockProbeIntervalMillis        int    `yaml:"lockProbeIntervalMillis"`
	InitialPostWriteIntervalSecond int    `yaml:"initialPostWriteIntervalSecond"`
	RestoreSupportsLock            bool   `yaml:"restoreSupportsLock"`
	LockFilePath                   string `yaml:"lockFilePath"`
}

type AutoDetect struct {
	PodCIDR   string `yaml:"podCIDR"`
	ClusterIP bool   `yaml:"clusterIP"`
	NodeIP    bool   `yaml:"nodeIP"`
}

// LoadConfig loads the configuration
func LoadConfig(isAgent bool) (*Config, error) {
	var ver iptables.Version
	var err error
	var restoreSupportsLock bool

	if isAgent {
		ver, err = iptables.GetVersion()
		if err != nil {
			return nil, err
		}
		restoreSupportsLock = ver.Compare(iptables.Version{Major: 1, Minor: 6, Patch: 2}) >= 0
	}

	config := &Config{
		EnvConfig: EnvConfig{
			LeaderElection:            true,
			LeaderElectionID:          "egressgateway",
			LeaderElectionLostRestart: false,
			HealthProbeBindAddress:    ":8788",
			MetricsBindAddress:        "0",
			GopsPort:                  0,
			WebhookPort:               8881,
			GolangMaxProcs:            -1,
			TLSCertDir:                "/etc/tls",
		},
		FileConfig: FileConfig{
			MaxNumberEndpointPerSlice: 100,
			IPTables: IPTables{
				RefreshIntervalSecond:   90,
				PostWriteIntervalSecond: 1,
				LockTimeoutSecond:       0,
				LockProbeIntervalMillis: 50,
				LockFilePath:            "/run/xtables.lock",
				RestoreSupportsLock:     restoreSupportsLock,
			},
			Mark: "0x26000000",
		},
	}

	// map environment variables to struct objects
	envKeysMap := &map[string]interface{}{}
	if err := mapstructure.Decode(config.EnvConfig, &envKeysMap); err != nil {
		return nil, err
	}
	for k := range *envKeysMap {
		if err := viper.BindEnv(k); err != nil {
			return nil, err
		}
	}

	err = viper.Unmarshal(&config.EnvConfig)
	if err != nil {
		return nil, err
	}

	// load file config from configMap
	if len(config.ConfigMapPath) > 0 {
		configmapBytes, err := os.ReadFile(config.ConfigMapPath)
		if nil != err {
			return nil, fmt.Errorf("failed to read ConfigMap file %v, error: %v", config.ConfigMapPath, err)
		}
		if err := yaml.Unmarshal(configmapBytes, &config.FileConfig); nil != err {
			return nil, fmt.Errorf("failed to parse ConfigMap data, error: %v", err)
		}
		if config.FileConfig.EnableIPv4 {
			_, ipn, err := net.ParseCIDR(config.FileConfig.TunnelIpv4Subnet)
			if err != nil {
				return nil, fmt.Errorf("failed to parse TunnelIpv4Subnet: %v", err)
			}
			config.FileConfig.TunnelIPv4Net = ipn
		}
		if config.FileConfig.EnableIPv6 {
			_, ipn, err := net.ParseCIDR(config.FileConfig.TunnelIpv6Subnet)
			if err != nil {
				return nil, fmt.Errorf("failed to parse TunnelIpv6Subnet: %v", err)
			}
			config.FileConfig.TunnelIPv6Net = ipn
		}
	}

	if config.FileConfig.IPTables.BackendMode == "auto" {
		config.FileConfig.IPTables.BackendMode = ver.BackendMode
	}

	config.Logger = logger.Config{
		UseDevMode: config.UseDevMode,
		WithCaller: config.WithCaller,
		Encoder:    config.Encoder,
	}
	level, err := strconv.ParseInt(config.Level, 10, 8)
	if err != nil {
		level, err := zap.ParseAtomicLevel(config.Level)
		if err != nil {
			return config, err
		}
		config.Logger.Level = level.Level()
	} else {
		// compatible with zap, the minimum is the maximum log level
		if level > 0 {
			level = 0 - level
		}
		config.Logger.Level = zapcore.Level(level)
	}

	list := config.FileConfig.AnnouncedInterfacesToExclude
	if len(list) > 0 {
		reg, err := regexp.Compile("(" + strings.Join(list, ")|(") + ")")
		if err != nil {
			return nil, err
		}
		config.FileConfig.AnnounceExcludeRegexp = reg
	}

	// load kube config
	config.KubeConfig, err = ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig, error: %v", config)
	}

	return config, nil
}
