package main

import (
	"context"

	"github.com/pkg/errors"
)

func istioInstalled(ctx context.Context) (installStatus, error) {
	ok, err := hasNamespace(ctx, "istio-system")
	if err != nil {
		return unknown, err
	} else if ok {
		return installed, nil
	}
	return notFound, nil
}

func podImageResolver(namespace, podPrefix, containerName string) versionFunc {
	return func(ctx context.Context) (versionInfo, error) {
		img, err := getPodImageByPrefix(ctx, namespace, podPrefix, containerName)
		if err != nil {
			return "", errors.Wrap(err, "failed to determine container image")
		}
		return versionInfo(img), nil
	}
}
