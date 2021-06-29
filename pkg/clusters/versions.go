/*
Copyright © 2021 Google

Licensed under the Apache License, version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package clusters

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/api/container/v1"
)

type Resource int

const (
	Node Resource = iota
	ControlPlane
)

// getVersions retrieves the default and valid versions based on the release channel.
func getVersions(sc *container.ServerConfig, cc string, res Resource) (string, []string) {
	var (
		defaultVersion = sc.DefaultClusterVersion
		validVersions  []string
	)
	if res == Node {
		validVersions = sc.ValidNodeVersions
	}
	if res == ControlPlane {
		validVersions = sc.ValidMasterVersions
	}
	if cc == "" {
		return defaultVersion, validVersions
	}
	for _, c := range sc.Channels {
		if c.Channel == cc {
			return c.DefaultVersion, c.ValidVersions
		}
	}
	// should not happen, but fallback just in case.
	return defaultVersion, validVersions
}

// IsFormatValid ensures that the version string is a valid GKE version or version alias.
// See: https://cloud.google.com/kubernetes-engine/versioning#specifying_cluster_version
func IsFormatValid(s string) error {
	if s == "-" || s == "latest" {
		return nil
	}

	split := strings.Split(s, "-")
	if len(split) > 2 {
		return fmt.Errorf("malformed version: %s", s)
	}

	kubernetes := split[0]
	ksplit := strings.Split(kubernetes, ".")
	if len(ksplit) < 2 || len(ksplit) > 3 {
		return fmt.Errorf("malformed version: %s", s)
	}
	_, err := strconv.Atoi(ksplit[0])
	if err != nil {
		return fmt.Errorf("malformed major version %s: %w", s, err)
	}
	_, err = strconv.Atoi(ksplit[1])
	if err != nil {
		return fmt.Errorf("malformed minor version %s: %w", s, err)
	}
	if len(ksplit) == 3 {
		_, err := strconv.Atoi(ksplit[2])
		if err != nil {
			return fmt.Errorf("malformed patch version %s: %w", s, err)
		}
	}

	if len(ksplit) != 3 && len(split) == 2 {
		return fmt.Errorf("malformed patch version %s: %w", s, err)
	}
	if len(split) != 2 {
		return nil
	}

	if !strings.HasPrefix(split[1], "gke.") {
		return fmt.Errorf("malformed GKE version: %s", s)
	}

	_, err = strconv.Atoi(strings.TrimPrefix(split[1], "gke."))
	if err != nil {
		return fmt.Errorf("malformed GKE version %s: %w", s, err)
	}

	return nil
}

// isVersionValid ensures that the desired version is not a downgrade and is in the list of valid versions.
// Note: version aliases are selected using a string prefix, and therefore the major/minor/patch/gke
// versions are not evaluated as integers.
//
// e.g. alias 1.20.1 will match 1.20.111-gke.1 before 1.20.11-gke.1 or 1.20.1-gke.1
func isVersionValid(desired, current, def string, valid []string) error {

	if desired == "-" && current == def {
		return fmt.Errorf("desired version %s (%s), current version: %s; cannot upgrade in-place", desired, def, current)
	}

	// Versions are in descending order, so select the first match and track the indexes.
	var (
		selectedVersion string
		selectedIndex   int
		currentIndex    int
	)
	for i, v := range valid {
		if strings.HasPrefix(v, desired) {
			selectedVersion = v
			selectedIndex = i
			break
		}
	}
	if selectedVersion == "" {
		return fmt.Errorf("desired version %s was not found; valid versions: %v", desired, valid)
	}

	for i, v := range valid {
		if v == current {
			currentIndex = i
			break
		}
	}

	if currentIndex <= selectedIndex {
		return fmt.Errorf("desired version %s must be newer than current version %s; valid versions: %v", desired, current, valid)
	}

	return nil
}
