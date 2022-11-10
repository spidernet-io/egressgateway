// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

// rbac marker:
// https://github.com/kubernetes-sigs/controller-tools/blob/master/pkg/rbac/parser.go
// https://book.kubebuilder.io/reference/markers/rbac.html

// +kubebuilder:rbac:groups=egressgateway.spidernet.io,resources=egressgatewaynodes;egressnodes;egressgatewaypolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=egressgateway.spidernet.io,resources=egressgatewaynodes/status;egressnodes/status;egressgatewaypolicies/status,verbs=get;update;patch

// +kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;watch;update;delete
// +kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=create;get;update
// +kubebuilder:rbac:groups="",resources=nodes;namespaces;endpoints;pods,verbs=get;list;watch;update

package v1
