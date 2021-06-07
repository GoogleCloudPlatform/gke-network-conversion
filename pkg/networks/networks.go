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
package networks

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"legacymigration/pkg"
	"legacymigration/pkg/clusters"
	"legacymigration/pkg/migrate"
	"legacymigration/pkg/operations"
)

type networkMigrator struct {
	projectID          string
	network            *compute.Network
	handler            operations.Handler
	clients            *pkg.Clients
	concurrentClusters uint16
	factory            func(c *container.Cluster) migrate.Migrator
}

func New(
	projectID string,
	network *compute.Network,
	handler operations.Handler,
	clients *pkg.Clients,
	concurrentClusters uint16,
	opts *clusters.Options) *networkMigrator {
	factory := func(c *container.Cluster) migrate.Migrator {
		return clusters.New(projectID, c, handler, clients, opts)
	}
	return &networkMigrator{
		projectID:          projectID,
		handler:            handler,
		network:            network,
		clients:            clients,
		concurrentClusters: concurrentClusters,
		factory:            factory,
	}
}

// Migrate performs the Network migration and then the cluster upgrades
func (m *networkMigrator) Migrate(ctx context.Context) error {
	parent := pkg.LocationPath(m.projectID, pkg.AnyLocation)
	resp, err := m.clients.Container.ListClusters(ctx, parent)
	if err != nil {
		return fmt.Errorf("error listing Clusters: %w", err)
	}

	return m.migrate(ctx, resp)
}

func (m *networkMigrator) migrate(ctx context.Context, cr *container.ListClustersResponse) error {
	if len(cr.MissingZones) > 0 {
		log.Warnf("Clusters.List response is missing zones: %v", cr.MissingZones)
	}

	filteredClusters := make([]*container.Cluster, 0)
	for _, c := range cr.Clusters {
		if c.Network == m.network.Name {
			filteredClusters = append(filteredClusters, c)
		}
	}

	// API returns error if no GCE resource with an internal IP (e.g. Cluster) is present on the Network:
	//  "No internal IP resources on the Network. This network does not need to be migrated."
	if len(filteredClusters) == 0 {
		log.Warnf("Network %q contains no clusters.", m.network.Name)
	}

	if err := m.migrateNetwork(ctx); err != nil {
		return err
	}

	cms := make([]migrate.Migrator, len(filteredClusters))
	for i, c := range filteredClusters {
		cms[i] = m.factory(c)
	}
	return m.migrateClusters(ctx, cms)
}

func (m *networkMigrator) migrateNetwork(ctx context.Context) error {
	if m.network.IPv4Range == "" {
		log.Infof("Network %q is already a VPC network.", m.network.Name)
		return nil
	}

	log.Infof("Switching legacy network %q to custom mode VPC network", m.network.Name)
	op, err := m.clients.Compute.SwitchToCustomMode(ctx, m.projectID, m.network.Name)
	if err != nil {
		original := err
		if op, err = m.clients.Compute.GetGlobalOperation(ctx, m.projectID, operations.ObtainID(err)); err != nil {
			return fmt.Errorf("error switching legacy network %q to custom mode VPC network: %w", m.network.Name, original)
		}
	}

	w := &ComputeOperation{
		ProjectID: m.projectID,
		Operation: op,
		Client:    m.clients.Compute,
	}
	if err := m.handler.Wait(ctx, w); err != nil {
		path := w.String()
		return fmt.Errorf("error waiting on Operation %s: %w", path, err)
	}

	log.Infof("Network %q switched to custom mode VPC network", m.network.Name)

	return nil
}

func (m *networkMigrator) migrateClusters(ctx context.Context, migrators []migrate.Migrator) error {
	log.Infof("Initiate upgrades for cluster(s) on network %q", m.network.Name)
	sem := make(chan struct{}, m.concurrentClusters)
	return migrate.Run(ctx, sem, migrators...)
}

type ComputeOperation struct {
	ProjectID string
	Operation *compute.Operation
	Client    pkg.ComputeService
}

func (o *ComputeOperation) String() string {
	return pkg.PathRegex.FindString(o.Operation.SelfLink)
}

func (o *ComputeOperation) poll(ctx context.Context) (operations.OperationStatus, error) {
	path := o.String()
	log.Debugf("Waiting for %s", path)
	var status operations.OperationStatus

	resp, err := o.Client.WaitOperation(ctx, o.ProjectID, o.Operation)
	if err != nil {
		return status, err
	}

	status = operationStatus(resp)
	log.Debugf("Operation %s status: %#v", path, status)
	return status, nil
}

func (o *ComputeOperation) IsFinished(ctx context.Context) (bool, error) {
	return operations.IsFinished(ctx, o.poll)
}

// operationStatus converts the status of a compute.Operation to a generic OperationStatus.
func operationStatus(op *compute.Operation) operations.OperationStatus {
	var errs string
	if op.Error != nil {
		var arr []string
		for _, e := range op.Error.Errors {
			arr = append(arr, e.Message)
		}
		errs = strings.Join(arr, "\n")
	}

	return operations.OperationStatus{
		Status: op.Status,
		Error:  errs,
	}
}
