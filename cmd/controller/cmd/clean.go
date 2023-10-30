// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	webhook "k8s.io/api/admissionregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean resources",
	Long:  "Clean resources with specified parameters.",
	Run: func(cmd *cobra.Command, args []string) {

		validate, err := cmd.Flags().GetString("validate")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		mutating, err := cmd.Flags().GetString("mutating")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("validate %s\nmutating %s\n", validate, mutating)
		err = clean(validate, mutating)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func clean(validate, mutating string) error {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	cli, err := client.New(cfg, client.Options{Scheme: schema.GetScheme()})
	if err != nil {
		return err
	}

	ctx := context.Background()

	vObj := &webhook.ValidatingWebhookConfiguration{}
	err = cli.Get(ctx, client.ObjectKey{Name: validate}, vObj)
	if err == nil {
		err := cli.Delete(ctx, vObj)
		if err != nil {
			return err
		}
	}

	mObj := &webhook.MutatingWebhookConfiguration{}
	err = cli.Get(ctx, client.ObjectKey{Name: mutating}, mObj)
	if err == nil {
		err := cli.Delete(ctx, mObj)
		if err != nil {
			return err
		}
	}

	list := new(egressv1.EgressTunnelList)
	err = cli.List(ctx, list)
	if err == nil {
		for _, item := range list.Items {
			item.Finalizers = make([]string, 0)
			err := cli.Update(ctx, &item)
			if err != nil {
				return err
			}
			err = cli.Delete(ctx, &item)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
