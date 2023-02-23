/*
Copyright 2022 Ciena Corporation.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ciena/outbound/internal/pkg/client"
	"github.com/ciena/outbound/internal/pkg/planner"
	"github.com/ciena/outbound/pkg/parallelize"
	"github.com/ciena/outbound/pkg/version"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultCallTimeout        = 15 * time.Second
	defaultUpdateWorkerPeriod = 15 * time.Second
)

type configSpec struct {
	KubeConfig         string
	Listen             string
	ShowVersion        bool
	ShowVersionAsJSON  bool
	Debug              bool
	CallTimeout        time.Duration
	Parallelism        int
	UpdateWorkerPeriod time.Duration
}

func getClients(kubeconfig string,
	log logr.Logger) (*kubernetes.Clientset, *client.SchedulePlannerClient, error,
) {
	var config *rest.Config

	var err error

	if kubeconfig == "" {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, nil, fmt.Errorf("could not get cluster config: %w", err)
		}
	} else {
		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			return nil, nil, fmt.Errorf("could not build config from flags: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get k8s clientset: %w", err)
	}

	plannerClient, err := client.NewSchedulePlannerClient(config, log.WithName("planner-client"))
	if err != nil {
		return nil, nil, fmt.Errorf("could not get planner client: %w", err)
	}

	return clientset, plannerClient, nil
}

func main() {
	var config configSpec

	flag.StringVar(&config.KubeConfig,
		"kubeconfig", "",
		"File that contains the client configuration to access Kubernetes cluster")
	flag.BoolVar(&config.Debug,
		"debug", true,
		"Display debug logging messages")
	flag.DurationVar(&config.CallTimeout,
		"call-timeout", defaultCallTimeout,
		"GRPC call timeout")
	flag.DurationVar(&config.UpdateWorkerPeriod,
		"update-worker-period", defaultUpdateWorkerPeriod,
		"The update period to retry resource updates on failures")

	flag.IntVar(&config.Parallelism,
		"paralleism", parallelize.DefaultParallelism,
		"Parallelism factor to run filter predicates to find eligible nodes for a pod")

	flag.BoolVar(&config.ShowVersion,
		"version", false,
		"Display version information and exit")
	flag.BoolVar(&config.ShowVersionAsJSON,
		"json", false,
		"If displaying version, do so a JSON object.")
	flag.StringVar(&config.Listen,
		"listen", ":7777",
		"Endpoint on which to listen for API requests to the planner.")
	flag.Parse()

	if config.ShowVersion {
		if config.ShowVersionAsJSON {
			//nolint:errchkjson
			bytes, _ := json.Marshal(version.Version())
			fmt.Fprintln(os.Stdout, string(bytes))
		} else {
			fmt.Fprintln(os.Stdout, version.Version().String())
		}

		os.Exit(0)
	}

	var log logr.Logger

	if config.Debug {
		zapLog, err := zap.NewDevelopment()
		if err != nil {
			panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
		}

		log = zapr.NewLogger(zapLog)
	} else {
		zapLog, err := zap.NewProduction()
		if err != nil {
			panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
		}

		log = zapr.NewLogger(zapLog)
	}

	k8sClient, plannerClient, err := getClients(config.KubeConfig, log.WithName("client"))
	if err != nil {
		log.Error(err, "Error initializing k8s and planner client interface")
		os.Exit(1)
	}

	podsetPlanner, err := planner.NewPlanner(planner.Options{
		CallTimeout:        config.CallTimeout,
		Parallelism:        config.Parallelism,
		UpdateWorkerPeriod: config.UpdateWorkerPeriod,
	},
		k8sClient,
		plannerClient,
		log.WithName("planner"),
	)
	if err != nil {
		panic(err.Error())
	}

	// Start the API server
	//nolint:exhaustruct
	api := &planner.APIServer{
		Listen:  config.Listen,
		Planner: podsetPlanner,
		Log:     log.WithName("planner").WithName("api"),
	}

	go func() {
		log.Info("start-api-server", "addr", config.Listen)
		log.Error(api.Run(), "api-server-failure")
		os.Exit(1)
	}()

	// wait forever
	select {}
}
