// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/pkg/errors"
)

type installStatus int

const (
	unknown installStatus = iota
	notFound
	installed
	failed
)

type versionInfo string

type detectFunc func(context.Context) (bool, error)
type versionFunc func(context.Context) (versionInfo, error)

type extension struct {
	name          string
	detectFn      detectFunc
	versionFn     versionFunc
	subcomponents []*extension

	result detectResult
}

type detectResult struct {
	status  installStatus
	version versionInfo
	error   error
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // TODO implement signal handling and call cancel()

	extensions := []*extension{
		&extension{
			name:     "istio",
			detectFn: detectByNamespacePrefix("istio-system"),
			subcomponents: []*extension{
				{
					name:      "pilot",
					detectFn:  detectByPod("istio-system", "istio-pilot-"),
					versionFn: versionFromDeploymentImage("istio-system", "istio-pilot", "discovery"),
				},
				{
					name:      "sidecar-injector",
					detectFn:  detectByPod("istio-system", "istio-sidecar-injector-"),
					versionFn: versionFromDeploymentImage("istio-system", "istio-sidecar-injector", ""),
				},
				{
					name:      "policy",
					detectFn:  detectByPod("istio-system", "istio-policy-"),
					versionFn: versionFromDeploymentImage("istio-system", "istio-policy", "mixer"),
				},
				{
					name:      "prometheus",
					detectFn:  detectByPod("istio-system", "prometheus-"),
					versionFn: versionFromDeploymentImage("istio-system", "prometheus", "prometheus"),
				},
			},
		},
		&extension{
			name:     "knative",
			detectFn: detectByNamespacePrefix("knative-"),
			subcomponents: []*extension{
				&extension{
					name:      "serving",
					detectFn:  detectByNamespace("knative-serving"),
					versionFn: versionFromDeploymentImage("knative-serving", "controller", ""),
				},
				&extension{
					name:      "build",
					detectFn:  detectByNamespace("knative-build"),
					versionFn: versionFromDeploymentImage("knative-build", "build-controller", ""),
				},
				&extension{
					name:      "eventing",
					detectFn:  detectByNamespace("knative-eventing"),
					versionFn: versionFromDeploymentImage("knative-eventing", "eventing-controller", ""),
				},
			},
		},
		&extension{
			name:      "helm-tiller",
			detectFn:  detectByPod("kube-system", "tiller-deploy-"),
			versionFn: versionFromDeploymentImage("kube-system", "tiller-deploy", "tiller"),
		},
	}
	if err := processExtensions(ctx, extensions); err != nil {
		log.Printf("WARN: failed to detect some extensions: %v", err)
	}
	printStatuses("", extensions)
}

func processExtensions(ctx context.Context, extensions []*extension) error {
	var wg sync.WaitGroup
	var outErr error
	for _, ex := range extensions {
		wg.Add(1)
		go func(e *extension) {
			defer wg.Done()
			if err := processExtension(ctx, e); err != nil {
				outErr = errors.Wrapf(err, "failed to process %q", e.name)
			}
		}(ex)
	}
	wg.Wait()
	return outErr
}

func processExtension(ctx context.Context, e *extension) error {
	if e.detectFn == nil {
		return errors.Errorf("extension %q has no detection function", e.name)
	}
	installStatus, err := e.detectFn(ctx)
	if err != nil {
		e.result.status = failed
		e.result.error = err
		return err // TODO: we don't return the err, so wanna continue processing
	}
	if installStatus {
		e.result.status = installed
	} else {
		e.result.status = notFound
	}

	if e.result.status != installed {
		return nil
	}
	// process subcomponents if any
	if len(e.subcomponents) > 0 {
		if err := processExtensions(ctx, e.subcomponents); err != nil {
			return errors.Wrapf(err, "failed to process subcomponents of %q", e.name)
		}
		return nil
	}

	if e.versionFn == nil {
		return errors.Errorf("extension %q has no version function", e.name)
	}
	version, err := e.versionFn(ctx)
	e.result.error = err
	if err != nil {
		e.result.status = failed
		e.result.error = err
		return err
	}
	e.result.version = version
	return nil
}

func statusText(r detectResult) string {
	switch r.status {
	case installed:
		return fmt.Sprintf("%s", r.version)
	case notFound:
		return "<not installed>"
	case unknown:
		return "???"
	case failed:
		return fmt.Sprintf("<error>: %s", r.error)
	default:
		return "<unhandled status>"
	}
}

func printStatuses(prefix string, extensions []*extension) {
	for _, e := range extensions {
		if len(e.subcomponents) == 0 {
			// leaf component
			fmt.Printf("%s- %s: %s", prefix, e.name, statusText(e.result))
		} else {
			// non-leaf component (if installed, do not print install status)
			fmt.Printf("%s- %s:", prefix, e.name)
			if e.result.status != installed {
				fmt.Printf(" %s", statusText(e.result))
			}
		}
		fmt.Println()

		if e.result.status == installed && len(e.subcomponents) > 0 {
			printStatuses(prefix+"  ", e.subcomponents)
		}
	}
}
