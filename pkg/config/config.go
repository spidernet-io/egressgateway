// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"go.uber.org/zap"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Config struct {
	// From environment
	EnvConfig

	// FileConfig from configmap
	FileConfig FileConfig

	// KubeConfig kubeconfig
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
	LogLevel                  string `mapstructure:"LOG_LEVEL"`
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
	EnableIPv4      bool `yaml:"enableIPv4"`
	EnableIPv6      bool `yaml:"enableIPv6"`
	StartRouteTable int  `yaml:"startRouteTable"`
	// ["auto", "legacy", "nft"]
	IptablesMode string `yaml:"iptablesMode"`
	// ["iptables", "ebpf"]
	DatapathMode     string `yaml:"datapathMode"`
	TunnelIpv4Subnet string `yaml:"tunnelIpv4Subnet"`
	TunnelIpv6Subnet string `yaml:"tunnelIpv6Subnet"`
	TunnelInterface  string `yaml:"tunnelInterface"`
	ForwardMethod    string `yaml:"forwardMethod"`

	VxlanID      int `yaml:"vxlanID"`
	VxlanUdpPort int `yaml:"vxlanUdpPort"`
}

// LoadConfig loads the configuration
func LoadConfig() (*Config, error) {
	config := &Config{
		EnvConfig: EnvConfig{
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
		FileConfig: FileConfig{},
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

	err := viper.Unmarshal(&config.EnvConfig)
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
	}

	// load kubeconfig
	config.KubeConfig, err = ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig, error: %v", config)
	}

	return config, nil
}
