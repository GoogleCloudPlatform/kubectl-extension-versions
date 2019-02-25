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
