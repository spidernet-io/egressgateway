// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

var vipCmd = &cobra.Command{
	Use:   "vip",
	Short: "vip resources",
	Long:  "vip resources with specified parameters.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

type DisplayEgressIPData struct {
	IPv4          string
	IPv6          string
	Node          string
	EgressGateway string
}

var egressGatewayName, vipAddress, targetNode string

var moveCmd = &cobra.Command{
	Use:   "moveEgressIP --egressGatewayName <egress-gateway-name> --vip <vip-address> --targetNode <node-name>",
	Short: "Move a VIP to a new node",
	Long:  `Move a Egress IP to the specified target node.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if egressGatewayName == "" || vipAddress == "" || targetNode == "" {
			fmt.Println("Error: egressGatewayName, vip address, and target node name must be specified")
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Moving VIP %s to node %s...\n", vipAddress, targetNode)
		err := moveEgressIP(egressGatewayName, vipAddress, targetNode)
		if err != nil {
			cmd.PrintErr("Move failed: ", err)
			os.Exit(1)
		}
	},
}

func moveEgressIP(egressGatewayName string, vipAddress, targetNode string) error {
	kubeConfig, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	cli, err := client.New(kubeConfig, client.Options{Scheme: schema.GetScheme()})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	egw := egressv1.EgressGateway{}
	err = cli.Get(ctx, types.NamespacedName{Name: egressGatewayName}, &egw)
	if err != nil {
		return fmt.Errorf("failed to get EgressGateway: %w", err)
	}

	eipPair, err := findAndRemoveEIP(&egw, vipAddress, targetNode)
	if err != nil {
		return err
	}
	if eipPair == nil {
		fmt.Printf("Egress IP %s is currently on node %s, with no change.\n", vipAddress, targetNode)
		return nil
	}

	if err := moveEIPToNode(&egw, targetNode, eipPair); err != nil {
		return err
	}
	if err := cli.Status().Update(ctx, &egw); err != nil {
		return fmt.Errorf("failed to update EgressGateway status: %w", err)
	}
	fmt.Printf("Successfully moved VIP %s to node %s\n", vipAddress, targetNode)
	return nil
}

// findAndRemoveEIP finds the EIP and removes it from the current node.
// Returns the found EIP and any error that occurs.
func findAndRemoveEIP(egw *egressv1.EgressGateway, vipAddress string, targetNode string) (*egressv1.Eips, error) {
	for nodeIndex, node := range egw.Status.NodeList {
		for eipIndex, eip := range node.Eips {
			if node.Name == targetNode {
				return nil, nil
			}

			if eip.IPv4 == vipAddress || eip.IPv6 == vipAddress {
				foundEIP := egw.Status.NodeList[nodeIndex].Eips[eipIndex]
				egw.Status.NodeList[nodeIndex].Eips = append(
					egw.Status.NodeList[nodeIndex].Eips[:eipIndex],
					egw.Status.NodeList[nodeIndex].Eips[eipIndex+1:]...,
				)
				return &foundEIP, nil
			}
		}
	}
	return nil, fmt.Errorf("VIP %s not found in any node of EgressGateway", vipAddress)
}

// moveEIPToNode moveEgressIP an EIP to the target node within the EgressGateway.
// Returns any error that occurs.
func moveEIPToNode(egw *egressv1.EgressGateway, targetNode string, eipPair *egressv1.Eips) error {
	found := false
	for i, node := range egw.Status.NodeList {
		if node.Name == targetNode {
			if string(egressv1.EgressTunnelReady) != node.Status {
				return fmt.Errorf("target node '%s' not ready, please select a ready node", targetNode)
			}

			egw.Status.NodeList[i].Eips = append(node.Eips, *eipPair)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("target node '%s' not found in EgressGateway", targetNode)
	}
	return nil
}
