// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var binName = filepath.Base(os.Args[0])

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   binName,
	Short: "egress gateway ctl",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	moveCmd.Flags().StringVarP(&egressGatewayName, "egressGatewayName", "", "", "Specify the name of the egress gateway")
	moveCmd.Flags().StringVarP(&vipAddress, "vip", "", "", "Specify the VIP address to moveEgressIP")
	moveCmd.Flags().StringVarP(&targetNode, "targetNode", "", "", "Specify the name of the node to moveEgressIP the VIP to")

	rootCmd.AddCommand(vipCmd)
	vipCmd.AddCommand(moveCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
