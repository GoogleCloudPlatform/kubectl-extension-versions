package main

import (
	"context"
	"log"

	"github.com/pkg/errors"
)

func versionInfoFromImage(image string) versionInfo {
	// can simplify gcr.io/[img]@sha256:[...] as versioned :tag?
	if isGCRHash(image) {
		i, err := resolveGCRHashToTag(image)
		if err != nil {
			log.Printf("WARN: failed to query tags for gcr image: %v", err)
		} else if i != "" {
			return versionInfo(image)
		}
	}
	return versionInfo(image) // fallback
}

func versionFromDeploymentImage(namespace, deploymentName, containerName string) versionFunc {
	return func(ctx context.Context) (versionInfo, error) {
		img, err := getPodImageByPrefix(ctx, namespace, deploymentName+"-", containerName)
		if err != nil {
			return "", errors.Wrap(err, "failed to determine container image")
		}
		return versionInfoFromImage(img), nil
	}
}
