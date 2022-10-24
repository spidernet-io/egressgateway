// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	pkgmetric "github.com/spidernet-io/egressgateway/pkg/metrics"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/view"
)

var (
	MetricCounterRequest    syncfloat64.Counter
	MetricGaugeEndpoint     syncfloat64.UpDownCounter
	MetricHistogramDuration syncfloat64.Histogram
)

var metricMapping = []pkgmetric.MetricMappingType{
	{P: &MetricCounterRequest, Name: "request_counts", Description: "the request counter"},
	{P: &MetricGaugeEndpoint, Name: "endpoint_number", Description: "the endpoint number"},
	{P: &MetricHistogramDuration, Name: "request_duration_seconds", Description: "the request duration histogram"},
}

// var globalMeter metric.Meter
func RunMetricsServer(meterName string) {
	logger := rootLogger.Named("metric")

	// View to customize histogram buckets
	customBucketsView, err := view.New(
		// MatchInstrumentName will match an instrument based on the its name.
		// This will accept wildcards of * for zero or more characters, and ? for
		// exactly one character. A name of "*" (default) will match all instruments.
		view.MatchInstrumentName("*duration*"),
		view.MatchInstrumentationScope(instrumentation.Scope{Name: meterName}),
		// With* to modify instruments
		view.WithSetAggregation(aggregation.ExplicitBucketHistogram{
			Boundaries: []float64{1, 10, 20, 50},
		}),
	)
	if err != nil {
		logger.Sugar().Fatalf("failed to generate view, reason=%v", err)
	}

	// globalMeter = pkgmetric.NewMetricsServer(meterName, globalConfig.MetricPort, metricMapping, customBucketsView, logger)
	pkgmetric.RunMetricsServer(globalConfig.EnableMetric, meterName, globalConfig.MetricPort, metricMapping, customBucketsView, logger)

}
