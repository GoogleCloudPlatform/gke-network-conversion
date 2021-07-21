/*
Copyright © 2021 Google

Licensed under the Apache License, Version 2.0 (the "License");
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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"legacymigration/pkg"

	"google.golang.org/api/container/v1"
)

type Resource int

const (
	Node Resource = iota
	ControlPlane

	// This is the maximum version skew allowed for auto-upgrade clusters.
	// This is used rather than the Kubernetes version skew (2).
	MaxVersionSkew = 1

	DefaultVersion = "-"
	LatestVersion  = "latest"
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
	return defaultVersion, validVersions
}

// IsFormatValid ensures that the version string is a valid GKE version or version alias.
// See: https://cloud.google.com/kubernetes-engine/versioning#specifying_cluster_version
func IsFormatValid(s string) error {
	if s == "" {
		return errors.New("malformed version: version must not be empty")
	}
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
	major, err := strconv.Atoi(ksplit[0])
	if err != nil {
		return fmt.Errorf("malformed major version %s: %w", s, err)
	}
	if major != 1 {
		return fmt.Errorf("not compatible with major versions other than 1: version %s", s)
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

// isUpgrade ensures that the desired version is not a downgrade and is in the list of valid versions.
func isUpgrade(desired, current string, valid []string, allowInPlace bool) error {
	if len(valid) == 0 {
		// Should not happen, but protects from out-of-bounds error.
		return fmt.Errorf("list of valid versions is empty: %v", valid)
	}

	var (
		desiredIndex = -1
		currentIndex = -1
	)
	for i, v := range valid {
		if v == desired {
			desiredIndex = i
		}
		if v == current {
			currentIndex = i
		}
	}
	if desiredIndex == -1 {
		return fmt.Errorf("desired version %s was not found; valid versions: %v", desired, valid)
	}
	if currentIndex == -1 {
		// The defalt version is no longer a valid version.
		// This can happen to node pools with auto-upgrade disabled.
		return nil
	}
	if allowInPlace && currentIndex == desiredIndex {
		return nil
	}
	if currentIndex <= desiredIndex {
		return fmt.Errorf("desired version %s must be newer than current version %s; valid versions: %v", desired, current, valid)
	}

	return nil
}

// IsWithinVersionSkew ensures that the node and control plane versions are within version skew.
// This helps avoid version skew API errors, e.g.:
//  `node version "x" must be within one minor version of master version "y"`
//
// Versions must be in the form "1\.x.*".
//
// Note: allowed GKE version skew depends on whether the cluster is using a release channel.
//  This method uses the release channel version skew value (1 minor version).
func IsWithinVersionSkew(npVersion, cpVersion string, allowedSkew int) error {
	npMinor, err := GetMinorVersion(npVersion)
	if err != nil {
		return err
	}
	cpMinor, err := GetMinorVersion(cpVersion)
	if err != nil {
		return err
	}

	diff := cpMinor - npMinor
	if diff < 0 {
		return fmt.Errorf("desired node version %s minor version (%d) cannot be greater than desired control plane version %s minor version (%d)",
			npVersion, npMinor, cpVersion, cpMinor)
	}
	if diff > allowedSkew {
		return fmt.Errorf("desired node version %s must be no less than %d minor versions from the desired control plane version %s",
			npVersion, allowedSkew, cpVersion)
	}

	return nil
}

// resolveVersion converts the desired version (alias) to a specific GKE version.
//
// Example(s):
//  1.21 -> 1.21.x-gke.y
//  "-"  -> 1.x.y-gke.z
func resolveVersion(desired, def string, valid []string) (string, error) {
	if len(valid) == 0 {
		// Should not happen, but protects from out-of-bounds error.
		return "", fmt.Errorf("list of valid versions is empty: %v", valid)
	}
	if def == "" {
		// Should not happen, but protects from unforeseen input return values.
		return "", fmt.Errorf("default version is missing: desired version %s", desired)
	}

	if desired == DefaultVersion {
		return def, nil
	}
	if desired == LatestVersion {
		return valid[0], nil
	}

	// Versions are in descending order, so select the first match.
	for _, v := range valid {
		if strings.HasPrefix(v, desired) {
			return v, nil
		}
	}

	return "", fmt.Errorf("desired version %q could not be resolved; valid versions: %v", desired, valid)
}

// GetMinorVersion gets the k8s minor version as an int.
// GetMinorVersion assumes validation has already been performed on the input.
func GetMinorVersion(v string) (int, error) {
	split := strings.Split(v, ".")
	return strconv.Atoi(split[1])
}

// getReleaseChannel returns the release channel if present.
// Otherwise, it returns unspecified.
func getReleaseChannel(rc *container.ReleaseChannel) string {
	if rc != nil {
		return rc.Channel
	}
	return pkg.Unspecified
}
