package main

import (
	"context"

	"github.com/pkg/errors"
)

var (
	istioInstalled = detectByNamespace("istio-system")
)

func podImageResolver(namespace, podPrefix, containerName string) versionFunc {
	return func(ctx context.Context) (versionInfo, error) {
		img, err := getPodImageByPrefix(ctx, namespace, podPrefix, containerName)
		if err != nil {
			return "", errors.Wrap(err, "failed to determine container image")
		}
		return versionInfoFromImage(img), nil
	}
}
