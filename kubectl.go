package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var (
	namespaces []string
	nsLock     sync.Once

	podList  []pod
	podsLock sync.Once
)

type pod struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Containers []struct {
			Name  string `json:"name"`
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

func hasNamespaceWithPrefix(ctx context.Context, prefix string) (bool, error) {
	ns, err := getNamespaces(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to get namespaces")
	}
	for _, n := range ns {
		if strings.HasPrefix(n, prefix) {
			return true, nil
		}
	}
	return false, nil
}

func hasNamespace(ctx context.Context, s string) (bool, error) {
	list, err := getNamespaces(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to get namespaces")
	}
	for _, n := range list {
		if n == s {
			return true, nil
		}
	}
	return false, nil
}

func detectByNamespace(ns string) detectFunc {
	return func(ctx context.Context) (installStatus, error) {
		ok, err := hasNamespace(ctx, ns)
		if err != nil {
			return failed, err
		} else if ok {
			return installed, nil
		}
		return notFound, nil
	}
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
		podList = v.Items
	})
	return podList, errOut
}

func hasPodsByPrefix(ctx context.Context, namespace, podPrefix string) (bool, error) {
	pods, err := getPods(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to get pods")
	}
	for _, p := range pods {
		if p.Metadata.Namespace == namespace && strings.HasPrefix(p.Metadata.Name, podPrefix) {
			return true, nil
		}
	}
	return false, nil
}

func detectByPod(namespace, podPrefix string) detectFunc {
	return func(ctx context.Context) (installStatus, error) {
		ok, err := hasPodsByPrefix(ctx, namespace, podPrefix)
		if err != nil {
			return failed, err
		}
		if ok {
			return installed, nil
		}
		return notFound, nil
	}
}

func getPodImageByPrefix(ctx context.Context, namespace, podPrefix, containerName string) (string, error) {
	pods, err := getPods(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get pods")
	}
	var p *pod
	for _, pp := range pods {
		if pp.Metadata.Namespace == namespace && strings.HasPrefix(pp.Metadata.Name, podPrefix) {
			p = &pp
			break
		}
	}
	if p == nil {
		return "", errors.Errorf("no pod found with \"%s/%s\" prefix", namespace, podPrefix)
	}
	if len(p.Spec.Containers) == 1 {
		return p.Spec.Containers[0].Image, nil
	}
	if containerName == "" {
		return "", errors.Errorf("pod %s has %d containers, could not disambiguate (containerName filter not given)", p.Metadata.Name, len(p.Spec.Containers))
	}
	for _, c := range p.Spec.Containers {
		if c.Name == containerName {
			return c.Image, nil
		}
	}
	return "", errors.Errorf("could not find container name %q in pod %s/%s", containerName, p.Metadata.Namespace, p.Metadata.Name)
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
