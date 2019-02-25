package main

import (
	"context"
	"strings"

	"github.com/pkg/errors"
)

func knativeInstalled(ctx context.Context) (installStatus, error) {
	ok, err := hasNamespaceWithPrefix(ctx, "knative-")
	if err != nil {
		return unknown, err
	} else if ok {
		return installed, nil
	}
	return notFound, nil
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
			return unknown, err
		} else if ok {
			return installed, nil
		}
		return notFound, nil
	}
}
