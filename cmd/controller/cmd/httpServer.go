// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/spidernet-io/egressgateway/api/v1/server"
	"github.com/spidernet-io/egressgateway/api/v1/server/restapi"
	"github.com/spidernet-io/egressgateway/api/v1/server/restapi/healthy"
	"go.uber.org/zap"
)

// ---------- readiness Healthy Handler
type readinessHealthyHandler struct {
	logger *zap.Logger
}

func (s *readinessHealthyHandler) Handle(r healthy.GetHealthyReadinessParams) middleware.Responder {
	// return healthy.NewGetHealthyReadinessInternalServerError()
	return healthy.NewGetHealthyReadinessOK()
}

// ---------- liveness Healthy Handler
type livenessHealthyHandler struct {
	logger *zap.Logger
}

func (s *livenessHealthyHandler) Handle(r healthy.GetHealthyLivenessParams) middleware.Responder {
	return healthy.NewGetHealthyLivenessOK()
}

// ---------- startup Healthy Handler
type startupHealthyHandler struct {
	logger *zap.Logger
}

func (s *startupHealthyHandler) Handle(r healthy.GetHealthyStartupParams) middleware.Responder {

	return healthy.NewGetHealthyStartupOK()
}

// ====================

func SetupHttpServer() {
	logger := rootLogger.Named("http")

	if globalConfig.HttpPort == 0 {
		logger.Sugar().Warn("http server is disabled")
		return
	}
	logger.Sugar().Infof("setup http server at port %v", globalConfig.HttpPort)

	spec, err := loads.Embedded(server.SwaggerJSON, server.FlatSwaggerJSON)
	if err != nil {
		logger.Sugar().Fatalf("failed to load Swagger spec, reason=%v ", err)
	}

	api := restapi.NewHTTPServerAPIAPI(spec)
	api.Logger = func(s string, i ...interface{}) {
		logger.Sugar().Infof(s, i)
	}

	// setup route
	api.HealthyGetHealthyReadinessHandler = &readinessHealthyHandler{logger: logger.Named("route: readiness health")}
	api.HealthyGetHealthyLivenessHandler = &livenessHealthyHandler{logger: logger.Named("route: liveness health")}
	api.HealthyGetHealthyStartupHandler = &startupHealthyHandler{logger: logger.Named("route: startup health")}

	//
	srv := server.NewServer(api)
	srv.EnabledListeners = []string{"http"}
	// srv.EnabledListeners = []string{"unix"}
	// srv.SocketPath = "/var/run/http-server-api.sock"

	// dfault to listen on "0.0.0.0" and "::1"
	// srv.Host = "0.0.0.0"
	srv.Port = int(globalConfig.HttpPort)
	srv.ConfigureAPI()

	go func() {
		e := srv.Serve()
		s := "http server break"
		if e != nil {
			s += fmt.Sprintf(" reason=%v", e)
		}
		logger.Fatal(s)
	}()

}
