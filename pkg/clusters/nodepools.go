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
	"context"
	"regexp"
	"strings"

	"google.golang.org/api/container/v1"
)

var (
	// Extract the location and name from an instance group manager parent path or URL.
	instanceGroupManagerRegex = regexp.MustCompile(`projects/[^/]+/(?:zones|regions)/([^/]+)/instanceGroupManagers/([^/]+)$`)
)

type nodePoolMigrator struct {
	*clusterMigrator
	nodePool *container.NodePool
}

func NewNodePool(
	clusterMigrator *clusterMigrator,
	nodePool *container.NodePool) *nodePoolMigrator {
	return &nodePoolMigrator{
		clusterMigrator: clusterMigrator,
		nodePool:        nodePool,
	}
}

// Migrate performs a NodePool upgrade is deemed necessary.
func (m *nodePoolMigrator) Migrate(ctx context.Context) error {

	return nil
}

// getName extracts the name portion of a resource's parent string
// e.g. getName("projects/x/locations/y/resources/z") -> "z"
func getName(path string) string {
	s := strings.Split(path, "/")
	return s[len(s)-1]
}

func parseSubmatch(re *regexp.Regexp, s string) ([]string, bool) {
	res := re.FindStringSubmatch(s)
	if res != nil {
		return res, true
	}
	return res, false
}
