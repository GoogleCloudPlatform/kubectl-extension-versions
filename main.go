package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"sync"

	"github.com/pkg/errors"
)

var (
	namespaces   []string
	namespacesMu sync.Once
)

type installStatus int

const (
	unknown installStatus = iota
	notFound
	installed
	failed
)

type versionInfo string

type detectResult struct {
	status      installStatus
	version     versionInfo
	detectError error
}

type detectFunc func(context.Context) (installStatus, versionInfo, error)

type extension struct {
	f             detectFunc
	subcomponents map[string]extension

	result detectResult
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // TODO implement signal handling and call cancel()

	extensions := map[string]extension{
		"istio": extension{},
		"knative": extension{
			subcomponents: map[string]extension{
				"build":    extension{},
				"serving":  extension{},
				"eventing": extension{},
			},
		},
	}
	detectExtensions(ctx, extensions)
}

func detectExtensions(ctx context.Context, extensions map[string]extension) {
	for k, ex := range extensions {

	}
}

func detectExtension(ctx context.Context, e *extension) error {
	if len(e.subcomponents) > 0 {
		return errors.New("encountered subcomponents")
	}
	status, version, err := e.f(ctx)
	if err != nil {
		return err
	}
	e.result = detectResult{
		status:      status,
		version:     version,
		detectError: err, // TODO(ahmetb) remove this field, as we raise the err
	}
	return nil
}

func getNamespaces(ctx context.Context) ([]string, error) {
	var out []string
	var errOut error

	namespacesMu.Do(func() {

		j, err := execKubectl(ctx, "get", "namespaces", "-o=json")
		if err != nil {
			errOut = err
			return
		}
		var v struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(j, &v); err != nil {
			errOut = errors.Wrap(err, "decoding json failed")
			return
		}
		for _, vv := range v.Items {
			out = append(out, vv.Metadata.Name)
		}
	})

	return out, errOut
}

func execKubectl(ctx context.Context, args ...string) ([]byte, error) {
	var stdout, stderr, combined bytes.Buffer

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdout = io.MultiWriter(&stdout, &combined)
	cmd.Stderr = io.MultiWriter(&stderr, &combined)
	if err := cmd.Run(); err != nil {
		return nil, errors.Errorf("kubectl command failed (%s). output=%s", err, combined.String())
	}
	return stdout.Bytes(), nil
}
