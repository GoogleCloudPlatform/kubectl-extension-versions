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

type detectFunc func(context.Context) (installStatus, error)
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
			name:     "knative",
			detectFn: knativeInstalled,
			subcomponents: []*extension{
				&extension{
					name:     "build",
					detectFn: detectByNamespace("knative-build")},
				&extension{
					name:     "serving",
					detectFn: detectByNamespace("knative-build")},
				&extension{
					name:     "eventing",
					detectFn: detectByNamespace("knative-eventing")},
			},
		},
	}
	if err := processExtensions(ctx, extensions); err != nil {
		log.Fatal(err)
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
				outErr = err
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
	status, err := e.detectFn(ctx)
	e.result = detectResult{
		status: status,
		error:  err,
	}
	if err != nil {
		return nil // TODO: we don't return the err, so wanna continue parsing
	}

	if status != installed {
		return nil
	}
	if len(e.subcomponents) == 0 {
		if e.versionFn == nil {
			return errors.Errorf("extension %q has no version function", e.name)
		}
		version, err := e.versionFn(ctx)
		e.result.error = err
		if err != nil {
			e.result.error = err
			return err
		}
		e.result.version = version
	} else {
		if err := processExtensions(ctx, e.subcomponents); err != nil {
			return errors.Wrapf(err, "failed to process subcomponents of %s", e.name)
		}
	}
	return nil
}

func printStatuses(prefix string, extensions []*extension) {
	for _, e := range extensions {
		fmt.Printf("%s- %s: %d (%s) (%v)\n", prefix, e.name, e.result.status, e.result.version, e.result.error)
		if len(e.subcomponents) > 0 {
			printStatuses(prefix+"  ", e.subcomponents)
		}
	}
}
