// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"net"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/spidernet-io/egressgateway/pkg/iptables"
)

const ForwardMethodActiveActive = "active-active"

type Config struct {
	// From environment
	EnvConfig

	// FileConfig from configmap
	FileConfig FileConfig

	KubeConfig *rest.Config
}

func (cfg *Config) PrintPrettyConfig(zap *zap.Logger) {
	zap.Sugar().Info("env config list:")
	envKeysMap := &map[string]interface{}{}
	if err := mapstructure.Decode(cfg.EnvConfig, &envKeysMap); err != nil {
		panic(err)
	}
	for k, v := range *envKeysMap {
		zap.Sugar().Infof("%s=%v", k, v)
	}
}

type EnvConfig struct {
	NodeName                  string `mapstructure:"NODE_NAME"`
	LogLevel                  string `mapstructure:"LOG_LEVEL"`
	KLOGLevel                 int    `mapstructure:"KLOG_LEVEL"`
	LeaderElection            bool   `mapstructure:"LEADER_ELECTION"`
	LeaderElectionNamespace   string `mapstructure:"LEADER_ELECTION_NAMESPACE"`
	LeaderElectionID          string `mapstructure:"LEADER_ELECTION_ID"`
	LeaderElectionLostRestart bool   `mapstructure:"LEADER_ELECTION_LOST_RESTART"`
	MetricsBindAddress        string `mapstructure:"METRICS_BIND_ADDRESS"`
	HealthProbeBindAddress    string `mapstructure:"HEALTH_PROBE_BIND_ADDRESS"`
	GopsPort                  int    `mapstructure:"GOPS_PORT"`
	WebhookPort               int    `mapstructure:"WEBHOOK_PORT"`
	PyroscopeServerAddr       string `mapstructure:"PYROSCOPE_SERVER_ADDR"`
	PodName                   string `mapstructure:"POD_NAME"`
	PodNamespace              string `mapstructure:"POD_NAMESPACE"`
	GolangMaxProcs            int32  `mapstructure:"GOLANG_MAX_PROCS"`
	TLSCertDir                string `mapstructure:"TLS_CERT_DIR"`
	ConfigMapPath             string `mapstructure:"CONFIGMAP_PATH"`
}

type FileConfig struct {
	EnableIPv4         bool     `yaml:"enableIPv4"`
	EnableIPv6         bool     `yaml:"enableIPv6"`
	StartRouteTable    int      `yaml:"startRouteTable"`
	IPTables           IPTables `yaml:"iptables"`
	DatapathMode       string   `yaml:"datapathMode"`
	TunnelIpv4Subnet   string   `yaml:"tunnelIpv4Subnet"`
	TunnelIpv6Subnet   string   `yaml:"tunnelIpv6Subnet"`
	TunnelIPv4Net      *net.IPNet
	TunnelIPv6Net      *net.IPNet
	TunnelDetectMethod string `yaml:"tunnelDetectMethod"`
	ForwardMethod      string `yaml:"forwardMethod"`
	VXLAN              VXLAN  `yaml:"vxlan"`
}

const (
	TunnelInterfaceDefaultRoute = "defaultRouteInterface"
	TunnelInterfaceSpecific     = "interface="
)

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
			KLOGLevel:                 2,
			LeaderElection:            true,
			LeaderElectionID:          "spider-egress-gateway",
			LeaderElectionLostRestart: false,
			HealthProbeBindAddress:    ":8788",
			MetricsBindAddress:        "0",
			GopsPort:                  0,
			WebhookPort:               8881,
			GolangMaxProcs:            -1,
			TLSCertDir:                "/etc/tls",
		},
		FileConfig: FileConfig{
			IPTables: IPTables{
				BackendMode:             ver.BackendMode,
				RefreshIntervalSecond:   90,
				PostWriteIntervalSecond: 1,
				LockTimeoutSecond:       0,
				LockProbeIntervalMillis: 50,
				LockFilePath:            "/run/xtables.lock",
				RestoreSupportsLock:     restoreSupportsLock,
			},
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

	// load kube config
	config.KubeConfig, err = ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig, error: %v", config)
	}

	return config, nil
}
