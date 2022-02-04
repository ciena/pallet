/*
Copyright 2021 Ciena Corporation.

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
	"github.com/ciena/outbound/internal/pkg/planner"
	"github.com/ciena/outbound/pkg/version"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

type configSpec struct {
	KubeConfig        string
	Listen            string
	ShowVersion       bool
	ShowVersionAsJSON bool
	Debug             bool
}

func k8sClient(kubeconfig string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	} else {
		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func main() {
	var config configSpec

	flag.StringVar(&config.KubeConfig,
		"kubeconfig", "",
		"File that contains the client configuration to access Kubernetes cluster")
	flag.BoolVar(&config.Debug,
		"debug", true,
		"Display debug logging messages")
	flag.BoolVar(&config.ShowVersion,
		"version", false,
		"Display version information and exit")
	flag.BoolVar(&config.ShowVersionAsJSON,
		"json", false,
		"If displaying version, do so a a JSON object.")
	flag.StringVar(&config.Listen,
		"listen", ":7777",
		"Endpoint on which to listen for API requests to the planner.")
	flag.Parse()

	if config.ShowVersion {
		if config.ShowVersionAsJSON {
			bytes, _ := json.Marshal(version.Version())
			fmt.Println(string(bytes))
		} else {
			fmt.Println(version.Version().String())
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

	clientset, err := k8sClient(config.KubeConfig)
	if err != nil {
		log.Error(err, "Error initializing k8s client interface")
		os.Exit(1)
	}

	podsetPlanner := planner.NewPlanner(planner.PlannerOptions{},
		clientset,
		log.WithName("planner"),
	)

	// Start the API server
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
