// Copyright 2020 Ciena Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package planner

import (
	"context"
	"fmt"
	"net"

	api "github.com/ciena/outbound/pkg/apis/planner"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// APIServer Structure anchor for the implementation of the telemetry API server.
type APIServer struct {
	Listen  string
	Log     logr.Logger
	Planner *PodSetPlanner
	api.UnimplementedSchedulePlannerServer
}

// Run Processing loop to listen for and dispatch incoming telemetry API requests.
func (a *APIServer) Run() error {
	lis, err := net.Listen("tcp", a.Listen)
	if err != nil {
		return fmt.Errorf("error listening on %s: %w", a.Listen, err)
	}

	grpcServer := grpc.NewServer()
	api.RegisterSchedulePlannerServer(grpcServer, a)
	reflection.Register(grpcServer)

	//nolint: wrapcheck
	return grpcServer.Serve(lis)
}

// BuildSchedulePlan builds a schedule plan based on the podset for the namespace.
func (a *APIServer) BuildSchedulePlan(ctx context.Context,
	in *api.SchedulePlanRequest) (*api.SchedulePlanResponse, error) {

	a.Log.Info("build-schedule-plan-request", "request", in)

	assignments, err := a.Planner.BuildSchedulePlan(ctx, in.Namespace, in.PodSet,
		in.ScheduledPod, in.EligibleNodes)
	if err != nil {
		//nolint:wrapcheck
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.SchedulePlanResponse{Assignments: assignments}, nil
}
