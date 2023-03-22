// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package iptables

import "github.com/prometheus/client_golang/prometheus"

func MetricCollectors() []prometheus.Collector {
	return []prometheus.Collector{
		countNumRestoreCalls,
		countNumRestoreErrors,
		countNumSaveCalls,
		countNumSaveErrors,
		gaugeNumChains,
		gaugeNumRules,
		countNumLinesExecuted,
		summaryLockAcquisitionTime,
		countLockRetries,
	}
}
