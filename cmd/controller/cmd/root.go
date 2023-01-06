// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/controller"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/profiling"
)

var binName = filepath.Base(os.Args[0])

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   binName,
	Short: "run egress gateway controller",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		cfg, err := config.LoadConfig(false)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		log := logger.NewStdoutLogger(cfg.LogLevel)
		cfg.PrintPrettyConfig(log.Named("config"))

		defer func() {
			if e := recover(); nil != e {
				log.Sugar().Errorf("expected panic: %v", e)
				debug.PrintStack()
				os.Exit(1)
			}
		}()

		log.Sugar().Info("start controller")
		err = run(ctx, log, cfg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func run(ctx context.Context, log *zap.Logger, config *config.Config) error {
	setupUtility(log.Named("debug"), config)

	ctl, err := controller.New(config, log)
	if err != nil {
		return err
	}

restart:
	err = ctl.Start(ctx)
	if err != nil {
		if err.Error() == "leader election lost" && config.LeaderElectionLostRestart {
			goto restart
		}
		return err
	}
	return nil
}

func setupUtility(log *zap.Logger, config *config.Config) {
	d := profiling.New(log)
	if config.GopsPort != 0 {
		d.RunGoPS(config.GopsPort)
	}

	if config.PyroscopeServerAddr != "" {
		d.RunPyroscope(config.PyroscopeServerAddr, config.PodName)
	}
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
