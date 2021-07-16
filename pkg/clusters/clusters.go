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

	"legacymigration/pkg"
	"legacymigration/pkg/migrate"
	"legacymigration/pkg/operations"

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/container/v1"
)

// Cluster and NodePoolOptions
type Options struct {
	ConcurrentNodePools        uint16
	DesiredControlPlaneVersion string
	DesiredNodeVersion         string
	InPlaceControlPlaneUpgrade bool
}

type clusterMigrator struct {
	projectID string
	cluster   *container.Cluster
	handler   operations.Handler
	clients   *pkg.Clients
	opts      *Options
	factory   func(n *container.NodePool) migrate.Migrator

	// Field(s) populated during Complete.
	resolvedDesiredControlPlaneVersion string
	serverConfig                       *container.ServerConfig
	releaseChannel                     string
	children                           []migrate.Migrator
}

func New(
	projectID string,
	cluster *container.Cluster,
	handler operations.Handler,
	clients *pkg.Clients,
	opts *Options) *clusterMigrator {
	m := &clusterMigrator{
		projectID: projectID,
		cluster:   cluster,
		handler:   handler,
		clients:   clients,
		opts:      opts,
	}
	m.factory = func(n *container.NodePool) migrate.Migrator {
		return NewNodePool(m, n)
	}
	return m
}

// Complete initializes this migrator instance.
func (m *clusterMigrator) Complete(ctx context.Context) error {
	resp, err := m.clients.Container.ListNodePools(ctx, m.ResourcePath())
	if err != nil {
		return fmt.Errorf("error retrieving NodePools for Cluster %s: %w", m.ResourcePath(), err)
	}
	path := pkg.LocationPath(m.projectID, m.cluster.Location)
	m.serverConfig, err = m.clients.Container.GetServerConfig(ctx, path)
	if err != nil {
		return fmt.Errorf("error retrieving ServerConfig for Cluster %s: %w", m.ResourcePath(), err)
	}

	m.releaseChannel = getReleaseChannel(m.cluster.ReleaseChannel)
	def, valid := getVersions(m.serverConfig, m.releaseChannel, ControlPlane)
	if m.opts.InPlaceControlPlaneUpgrade {
		m.resolvedDesiredControlPlaneVersion = m.cluster.CurrentMasterVersion
	} else {
		m.resolvedDesiredControlPlaneVersion, err = resolveVersion(m.opts.DesiredControlPlaneVersion, def, valid)
		if err != nil {
			return err
		}
	}

	m.children = make([]migrate.Migrator, len(resp.NodePools))
	for i, np := range resp.NodePools {
		m.children[i] = m.factory(np)
	}

	log.Infof("Initialize NodePool objects for Cluster %s", m.ResourcePath())
	sem := make(chan struct{}, m.opts.ConcurrentNodePools)
	return migrate.Complete(ctx, sem, m.children...)
}

// Validate confirms that this an any child migrators are valid.
func (m *clusterMigrator) Validate(ctx context.Context) error {
	_, valid := getVersions(m.serverConfig, m.releaseChannel, ControlPlane)
	if err := isUpgrade(m.resolvedDesiredControlPlaneVersion, m.cluster.CurrentMasterVersion, valid, true); err != nil {
		return fmt.Errorf("validation error for Cluster %s: %w", m.ResourcePath(), err)
	}

	log.Infof("Upgrade for Cluster %s is valid; desired: %q (%s), current: %s",
		m.ResourcePath(), m.opts.DesiredControlPlaneVersion, m.resolvedDesiredControlPlaneVersion, m.cluster.CurrentMasterVersion)
	log.Infof("Validate NodePool upgrade(s) for Cluster %s", m.ResourcePath())
	sem := make(chan struct{}, m.opts.ConcurrentNodePools)
	return migrate.Validate(ctx, sem, m.children...)
}

// Migrate performs upgrade on the Cluster
func (m *clusterMigrator) Migrate(ctx context.Context) error {
	if err := m.upgradeControlPlane(ctx); err != nil {
		return err
	}

	return m.upgradeNodePools(ctx)
}

func (m *clusterMigrator) upgradeControlPlane(ctx context.Context) error {
	if m.cluster.Subnetwork != "" {
		log.Infof("Cluster %s does not require control plane upgrade.", m.ResourcePath())
		return nil
	}

	req := &container.UpdateMasterRequest{
		Name:          m.ResourcePath(),
		MasterVersion: m.resolvedDesiredControlPlaneVersion,
	}

	log.Infof("Upgrading control plane for Cluster %q to version %q", req.Name, req.MasterVersion)

	op, err := m.clients.Container.UpdateMaster(ctx, req)
	if err != nil {
		original := err
		name := pkg.OperationsPath(m.projectID, m.cluster.Location, operations.ObtainID(err))
		if op, err = m.clients.Container.GetOperation(ctx, name); err != nil {
			return fmt.Errorf("error upgrading control plane for Cluster %s: %w", m.ResourcePath(), original)
		}
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

	log.Infof("Upgraded control plane for Cluster %q to version %q", req.Name, req.MasterVersion)

	resp, err := m.clients.Container.GetCluster(ctx, m.ResourcePath())
	if err != nil {
		return fmt.Errorf("unable to confirm subnetwork value for cluster %s: %w", m.ResourcePath(), err)
	}
	if resp.Subnetwork == "" {
		return fmt.Errorf("subnetwork field is empty for cluster %s", m.ResourcePath())
	}

	return nil
}

// upgradeNodePools upgrades all Nodes for a clusters.
// This is to ensure that the instance templates for the nodes
func (m *clusterMigrator) upgradeNodePools(ctx context.Context) error {
	log.Infof("Initiate NodePool upgrades for Cluster %s", m.ResourcePath())
	sem := make(chan struct{}, m.opts.ConcurrentNodePools)
	return migrate.Migrate(ctx, sem, m.children...)
}

// ResourcePath formats identifying information about the cluster.
func (m *clusterMigrator) ResourcePath() string {
	return pkg.ClusterPath(m.projectID, m.cluster.Location, m.cluster.Name)
}

type ContainerOperation struct {
	ProjectID string
	Path      string
	Client    pkg.ContainerService
}

func (o *ContainerOperation) String() string {
	return o.Path
}

func (o *ContainerOperation) poll(ctx context.Context) (operations.OperationStatus, error) {
	log.Debugf("Polling for %s", o.String())

	var status operations.OperationStatus

	resp, err := o.Client.GetOperation(ctx, o.Path)
	if err != nil {
		return status, fmt.Errorf("error retrieving Operation %s: %w", o.Path, err)
	}

	status = operationStatus(resp)

	log.Debugf("Operation %s status: %#v", o.Path, status)
	return status, nil
}

func (o *ContainerOperation) IsFinished(ctx context.Context) (bool, error) {
	return operations.IsFinished(ctx, o.poll)
}

func operationStatus(op *container.Operation) operations.OperationStatus {
	var msg string
	if op.Error != nil {
		msg = op.Error.Message
	}
	return operations.OperationStatus{
		Status: op.Status,
		Error:  msg,
	}
}
