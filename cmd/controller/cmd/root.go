// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/controller"
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
		cfg.PrintPrettyConfig()

		defer func() {
			if e := recover(); nil != e {
				fmt.Println(e)
				os.Exit(1)
			}
		}()

		err = run(ctx, cfg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func run(ctx context.Context, config *config.Config) error {
	ctl, err := controller.New(config)
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

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cleanCmd.Flags().String("validate", "", "Specify validate parameter")
	cleanCmd.Flags().String("mutating", "", "Specify mutating parameter")

	rootCmd.AddCommand(cleanCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
