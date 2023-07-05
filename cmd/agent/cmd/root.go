// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spidernet-io/egressgateway/pkg/agent"
	"github.com/spidernet-io/egressgateway/pkg/config"
	"os"
	"os/signal"
	"path/filepath"
)

var binName = filepath.Base(os.Args[0])

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   binName,
	Short: "run egress gateway agent",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		cfg, err := config.LoadConfig(true)
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
	svc, err := agent.New(config)
	if err != nil {
		return err
	}
	err = svc.Start(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
