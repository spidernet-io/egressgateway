// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

// +kubebuilder:rbac:groups=egressgateway.spidernet.io,resources=egressgateways;egressnodes;egressclustergatewaypolicies;egressgatewaypolicies;egressendpointslices;egressclusterinfos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=egressgateway.spidernet.io,resources=egressgateways/status;egressnodes/status;egressclustergatewaypolicies/status;egressgatewaypolicies/status;egressclusterinfos/status,verbs=get;update;patch

// +kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;watch;update;delete
// +kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=create;get;update
// +kubebuilder:rbac:groups="",resources=nodes;namespaces;endpoints;pods;services,verbs=get;list;watch;update

// +kubebuilder:rbac:groups=crd.projectcalico.org,resources=ippools,verbs=get;list;watch;create;update;patch;delete

package v1beta1
