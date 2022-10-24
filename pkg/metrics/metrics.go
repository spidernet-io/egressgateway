// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/view"
	"go.uber.org/zap"
	"net/http"
)

type MetricMappingType struct {
	P           interface{}
	Name        string
	Description string
}

func RegisterMetricInstance(metricMapping []MetricMappingType, meter metric.Meter, logger *zap.Logger) {

	for _, v := range metricMapping {
		switch v.P.(type) {
		case *syncfloat64.Counter:
			t, e := meter.SyncFloat64().Counter(v.Name, instrument.WithDescription(v.Description))
			if e != nil {
				logger.Sugar().Fatalf("failed to generate counter metric %v, reason=%v", v.Name, e)
			}
			// ctx: will not record metric if ctx.Err()!=nil
			r := v.P.(*syncfloat64.Counter)
			*r = t
			logger.Info("new counter metric: " + v.Name)

		case *syncfloat64.UpDownCounter:
			t, e := meter.SyncFloat64().UpDownCounter(v.Name, instrument.WithDescription(v.Description))
			if e != nil {
				logger.Sugar().Fatalf("failed to generate gauge metric %v, reason=%v", v.Name, e)
			}
			r := v.P.(*syncfloat64.UpDownCounter)
			*r = t
			logger.Info("new gauge metric: " + v.Name)

		case *syncfloat64.Histogram:
			t, e := meter.SyncFloat64().Histogram(v.Name, instrument.WithDescription(v.Description))
			if e != nil {
				logger.Sugar().Fatalf("failed to generate histogram metric %v, reason=%v", v.Name, e)
			}
			r := v.P.(*syncfloat64.Histogram)
			*r = t
			logger.Info("new histogram metric: " + v.Name)

		default:
			logger.Sugar().Fatalf("unsupported metric: %+v", v)
		}
	}

}

// example: https://github.com/open-telemetry/opentelemetry-go/blob/main/example/prometheus/main.go
// https://github.com/open-telemetry/opentelemetry-go/blob/main/example/view/main.go
func RunMetricsServer(enabled bool, meterName string, metricPort int32, metricMapping []MetricMappingType, histogramBucketsView view.View, logger *zap.Logger) metric.Meter {

	if !enabled {
		logger.Sugar().Infof("metric server '%v' is disabled, create a fake metric server ", meterName)
		globalMeter := metric.NewNoopMeterProvider().Meter(meterName)
		RegisterMetricInstance(metricMapping, globalMeter, logger)
		return globalMeter
	}

	logger.Sugar().Infof("metric server '%v' will listen on port %v", meterName, metricPort)

	// The exporter embeds a default OpenTelemetry Reader and
	// implements prometheus.Collector, allowing it to be used as
	// both a Reader and Collector.
	exporter, err := otelprometheus.New()
	if err != nil {
		logger.Sugar().Fatalf("failed to generate prometheus exporter, reason=%v", err)
	}

	// Default view for other instruments
	defaultView, err := view.New(view.MatchInstrumentName("*"))
	if err != nil {
		logger.Sugar().Fatalf("failed to generate view, reason=%v", err)
	}

	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(exporter, defaultView, histogramBucketsView))
	globalMeter := provider.Meter(meterName)

	http.Handle("/metrics", promhttp.Handler())

	RegisterMetricInstance(metricMapping, globalMeter, logger)

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", metricPort), nil)
		s := "metric server is down"
		if err != nil {
			s += fmt.Sprintf(" reason: %v", err)
		}
		logger.Fatal(s)
	}()
	return globalMeter
}
