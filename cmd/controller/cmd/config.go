// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"strconv"
)

type Config struct {
	// from env
	EnableMetric bool
	MetricPort   int32
	HttpPort     int32
	GopsPort     int32
	WebhookPort  int32

	PyroscopeServerAddress string

	PodName      string
	PodNamespace string

	GolangMaxProcs int32

	// from flags
	ConfigMapPath string

	TlsCaCertPath     string
	TlsServerCertPath string
	TlsServerKeyPath  string

	// from configmap
	Configmap ConfigmapConfig
}

type ConfigmapConfig struct {
	EnableIPv4      bool `yaml:"enableIPv4"`
	EnableIPv6      bool `yaml:"enableIPv6"`
	StartRouteTable int  `yaml:"startRouteTable"`
}

var globalConfig Config

type _envMapping struct {
	envName      string
	defaultValue string
	p            interface{}
}

var envMapping = []_envMapping{
	{"ENV_ENABLED_METRIC", "false", &globalConfig.EnableMetric},
	{"ENV_METRIC_HTTP_PORT", "", &globalConfig.MetricPort},
	{"ENV_HTTP_PORT", "8787", &globalConfig.HttpPort},
	{"ENV_GOPS_LISTEN_PORT", "", &globalConfig.GopsPort},
	{"ENV_WEBHOOK_PORT", "", &globalConfig.WebhookPort},
	{"ENV_PYROSCOPE_PUSH_SERVER_ADDRESS", "", &globalConfig.PyroscopeServerAddress},
	{"ENV_POD_NAME", "", &globalConfig.PodName},
	{"ENV_POD_NAMESPACE", "", &globalConfig.PodNamespace},
	{"ENV_GOLANG_MAXPROCS", "8", &globalConfig.GolangMaxProcs},
}

func init() {

	viper.AutomaticEnv()
	if t := viper.GetString("ENV_LOG_LEVEL"); len(t) > 0 {
		rootLogger = logger.NewStdoutLogger(t)
	} else {
		rootLogger = logger.NewStdoutLogger("")
	}

	logger := rootLogger.Named("config")
	// env built in the image
	if t := viper.GetString("ENV_VERSION"); len(t) > 0 {
		logger.Info("app version " + t)
	}
	if t := viper.GetString("ENV_GIT_COMMIT_VERSION"); len(t) > 0 {
		logger.Info("git commit version " + t)
	}
	if t := viper.GetString("ENV_GIT_COMMIT_TIMESTAMP"); len(t) > 0 {
		logger.Info("git commit timestamp " + t)
	}

	for n, v := range envMapping {
		m := v.defaultValue
		if t := viper.GetString(v.envName); len(t) > 0 {
			m = t
		}
		if len(m) > 0 {
			switch v.p.(type) {
			case *int32:
				if s, err := strconv.ParseInt(m, 10, 64); err == nil {
					r := envMapping[n].p.(*int32)
					*r = int32(s)
				} else {
					logger.Fatal("failed to parse env value of " + v.envName + " to int32, value=" + m)
				}
			case *string:
				r := envMapping[n].p.(*string)
				*r = m
			case *bool:
				if s, err := strconv.ParseBool(m); err == nil {
					r := envMapping[n].p.(*bool)
					*r = s
				} else {
					logger.Fatal("failed to parse env value of " + v.envName + " to bool, value=" + m)
				}
			default:
				logger.Sugar().Fatal("unsupported type to parse %v, config type=%v ", v.envName, reflect.TypeOf(v.p))
			}
		}

		logger.Info(v.envName + " = " + m)
	}

	// command flags
	globalFlag := rootCmd.PersistentFlags()
	globalFlag.StringVarP(&globalConfig.ConfigMapPath, "config-path", "C", "", "configmap file path")
	globalFlag.StringVarP(&globalConfig.TlsCaCertPath, "tls-ca-cert", "R", "", "ca file path")
	globalFlag.StringVarP(&globalConfig.TlsServerCertPath, "tls-server-cert", "T", "", "server cert file path")
	globalFlag.StringVarP(&globalConfig.TlsServerKeyPath, "tls-server-key", "Y", "", "server key file path")
	if e := viper.BindPFlags(globalFlag); e != nil {
		logger.Sugar().Fatalf("failed to BindPFlags, reason=%v", e)
	}
	printFlag := func() {
		logger.Info("config-path = " + globalConfig.ConfigMapPath)
		logger.Info("tls-ca-cert = " + globalConfig.TlsCaCertPath)
		logger.Info("tls-server-cert = " + globalConfig.TlsServerCertPath)
		logger.Info("tls-server-key = " + globalConfig.TlsServerKeyPath)

		// load configmap
		if len(globalConfig.ConfigMapPath) > 0 {
			configmapBytes, err := os.ReadFile(globalConfig.ConfigMapPath)
			if nil != err {
				logger.Sugar().Fatalf("failed to read configmap file %v, error: %v", globalConfig.ConfigMapPath, err)
			}
			if err := yaml.Unmarshal(configmapBytes, &globalConfig.Configmap); nil != err {
				logger.Sugar().Fatalf("failed to parse configmap data, error: %v", err)
			}
		}
	}
	cobra.OnInitialize(printFlag)

}
