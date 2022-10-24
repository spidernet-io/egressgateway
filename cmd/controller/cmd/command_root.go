// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"syscall"
)

var BinName = filepath.Base(os.Args[0])
var rootLogger *zap.Logger

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   BinName,
	Short: "short description",
	Run: func(cmd *cobra.Command, args []string) {

		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
		go func() {
			for s := range c {
				rootLogger.Sugar().Warnf("got signal=%+v \n", s)
			}
		}()

		defer func() {
			if e := recover(); nil != e {
				rootLogger.Sugar().Errorf("Panic details: %v", e)
				debug.PrintStack()
				os.Exit(1)
			}
		}()
		DaemonMain()
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	currentP := runtime.GOMAXPROCS(-1)
	rootLogger.Sugar().Infof("%v: default max golang procs %v \n", BinName, currentP)
	if currentP > int(globalConfig.GolangMaxProcs) {
		runtime.GOMAXPROCS(int(globalConfig.GolangMaxProcs))
		currentP = runtime.GOMAXPROCS(-1)
		rootLogger.Sugar().Infof("%v: change max golang procs %v \n", BinName, currentP)
	}

	if err := rootCmd.Execute(); err != nil {
		rootLogger.Fatal(err.Error())
	}
}
