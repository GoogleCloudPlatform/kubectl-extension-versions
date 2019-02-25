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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"regexp"
)

var (
	gcrSHA256Pattern = regexp.MustCompile(`^gcr.io\/.*@sha256:[0-9a-f]{64}$`) // TODO is this redundant given below
	gcrSHA256Group   = regexp.MustCompile(`^gcr.io\/(.*)@(sha256:[0-9a-f]{64})$`)
)

func isGCRHash(image string) bool { return gcrSHA256Pattern.MatchString(image) }

// resolveGCRHashToTag returns the image with IMAGE:TAG format if it can be
// resolved. If no tags are available, an empty string is returned. If multiple
// tags are available, first one that's not "latest" is returned.
func resolveGCRHashToTag(image string) (string, error) {
	groups := gcrSHA256Group.FindStringSubmatch(image)
	if len(groups) != 3 {
		return "", errors.Errorf("image %s cannot be parsed into repo/sha (got %d groups)", image, len(groups))
	}
	repo, hash := groups[1], groups[2]

	resp, err := http.Get(fmt.Sprintf("https://gcr.io/v2/%s/tags/list", repo))
	if err != nil {
		return "", errors.Wrapf(err, "failed to query tags from GCR for image %s", image)
	}
	defer resp.Body.Close()
	var v struct {
		Manifest map[string]struct {
			Tags []string `json:"tag"`
		} `json:"manifest"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", errors.Wrap(err, "failed to read and decode response body")
	}
	man, ok := v.Manifest[hash]
	if !ok {
		return "", errors.Wrapf(err, "hash %q not found in response manifest", hash)
	}
	if len(man.Tags) == 0 {
		return "", errors.Errorf("no tags found for gcr image %s", image)
	}
	// return the first tag that's not "latest"
	var tag string
	for _, t := range man.Tags {
		if t != "latest" {
			tag = t
			break
		}
	}
	if tag == "" {
		tag = man.Tags[0]
	}
	return fmt.Sprintf("gcr.io/%s:%s", repo, tag), nil
}
