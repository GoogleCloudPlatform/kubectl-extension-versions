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
	namespaces []string
	nsLock     sync.Once

	pods     []pod
	podsLock sync.Once
)

type pod struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Containers []struct {
			Image string `json:"image"`
		} `json:"containers"`
	} `json:"spec"`
}

func getNamespaces(ctx context.Context) ([]string, error) {
	var errOut error

	nsLock.Do(func() {
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
			namespaces = append(namespaces, vv.Metadata.Name)
		}
	})

	return namespaces, errOut
}

func getPods(ctx context.Context) ([]pod, error) {
	var errOut error
	podsLock.Do(func() {
		j, err := execKubectl(ctx, "get", "pods", "--all-namespaces", "-o=json")
		if err != nil {
			errOut = err
			return
		}
		var v struct{ Items []pod }
		if err := json.Unmarshal(j, &v); err != nil {
			errOut = errors.Wrap(err, "decoding json failed")
			return
		}
		pods = v.Items
	})
	return pods, errOut
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
