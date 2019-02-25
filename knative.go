package main

import (
	"context"
	"log"

	"github.com/pkg/errors"
)

func resolveKnativeComponentVersion(namespace, deployment string) versionFunc {
	return func(ctx context.Context) (versionInfo, error) {
		img, err := getPodImageByPrefix(ctx, namespace, deployment+"-", "")
		if err != nil {
			return "", errors.Wrap(err, "failed to determine container image")
		}
		return versionInfoFromImage(img), nil
	}
}

func versionInfoFromImage(image string) versionInfo {
	// can simplify gcr.io/[img]@sha256:[...] as versioned :tag?
	if isGCRHash(image) {
		i, err := resolveGCRHashToTag(image)
		if err == nil {
			return versionInfo(i)
		}
		log.Printf("WARN: failed to query tags for gcr image: %v", err)
	}

	// TODO what else?

	return versionInfo(image) // fallback
}
