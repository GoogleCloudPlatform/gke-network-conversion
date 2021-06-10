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
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
	"google.golang.org/api/container/v1"
	"legacymigration/pkg"
	"legacymigration/pkg/operations"
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
	required, err := m.requiresUpgrade(ctx)
	if err != nil {
		return fmt.Errorf("not upgrading NodePool %s: %w", m.NodePoolPath(), err)
	}
	if !required {
		log.Infof("Upgrade not required for NodePool %s; skipping.", m.NodePoolPath())
		return nil
	}

	log.Infof("Upgrading NodePool %s", m.NodePoolPath())

	return m.migrate(ctx)
}

func (m *nodePoolMigrator) migrate(ctx context.Context) error {
	npp := m.NodePoolPath()
	version := m.opts.DesiredNodeVersion
	if m.opts.InPlaceUpgrade {
		version = m.nodePool.Version
	}
	req := &container.UpdateNodePoolRequest{Name: npp, NodeVersion: version}
	log.Infof("Upgrading NodePool %s to version %q", npp, req.NodeVersion)
	op, err := m.clients.Container.UpdateNodePool(ctx, req)
	if err != nil {
		original := err
		if op, err = m.clients.Container.GetOperation(ctx, operations.ObtainID(err)); err != nil {
			return fmt.Errorf("error upgrading NodePool %s: %w", npp, original)
		}
	}

	if !m.opts.WaitForNodeUpgrade {
		log.Infof("Not waiting on upgrade for NodePool %s. To monitor, exec:\n\t"+
			"gcloud container operations wait %s", npp, op.Name)
		return nil
	}

	path := pkg.PathRegex.FindString(op.SelfLink)
	w := &ContainerOperation{
		ProjectID: m.projectID,
		Path:      path,
		Client:    m.clients.Container,
	}
	if err := m.handler.Wait(ctx, w); err != nil {
		return fmt.Errorf("error waiting on Operation %s: %w", path, err)
	}

	log.Infof("NodePool %s upgraded. ", path)
	return nil
}

// ClusterPath formats identifying information about the cluster.
func (m *nodePoolMigrator) NodePoolPath() string {
	return pkg.NodePoolPath(m.projectID, m.cluster.Location, m.cluster.Name, m.nodePool.Name)
}

// requiresUpgrade returns whether to upgrade the NodePool based on the desired state.
func (m *nodePoolMigrator) requiresUpgrade(ctx context.Context) (bool, error) {
	var (
		errors   error
		required bool
	)
	for _, url := range m.nodePool.InstanceGroupUrls {
		res := instanceGroupManagerRegex.FindStringSubmatch(url)
		if res == nil {
			errors = multierr.Append(errors, fmt.Errorf("unable to parse location and name information from InstanceGroup URL (%s) for NodePool %s", url, m.NodePoolPath()))
			continue
		}

		igm, err := m.clients.Compute.GetInstanceGroupManager(ctx, m.projectID, res[1], res[2])
		if err != nil {
			errors = multierr.Append(errors, fmt.Errorf("error retrieving InstanceGroupManagers (%s) for NodePool %s: %w", url, m.NodePoolPath(), err))
			continue
		}

		it, err := m.clients.Compute.GetInstanceTemplate(ctx, m.projectID, getName(igm.InstanceTemplate))
		if err != nil {
			errors = multierr.Append(errors, fmt.Errorf("error retrieving GetInstanceTemplateResp %s for NodePool %s: %w", igm.InstanceTemplate, m.NodePoolPath(), err))
			continue
		}
		missing := true
		for _, ni := range it.Properties.NetworkInterfaces {
			if ni.Subnetwork != "" {
				missing = false
				break
			}
		}
		if missing {
			required = true
			break
		}
	}

	if errors != nil && !required {
		return required, fmt.Errorf("error(s) encountered obtaining an InstanceTemplate for NodePool %s: %w", m.NodePoolPath(), errors)
	}
	if errors != nil {
		log.Infof("Error(s) retrieving InstanceTemplate(s) for NodePool %s: %v", m.NodePoolPath(), errors)
	}

	return required, nil
}

// getName extracts the name portion of a resource's parent string
// e.g. getName("projects/x/locations/y/resources/z") -> "z"
func getName(path string) string {
	s := strings.Split(path, "/")
	return s[len(s)-1]
}
