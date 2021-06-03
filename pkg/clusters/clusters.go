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

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/container/v1"
	"legacymigration/pkg"
	"legacymigration/pkg/migrate"
	"legacymigration/pkg/operations"
)

// Cluster and NodePoolOptions
type Options struct {
	ConcurrentNodePools        uint16
	WaitForNodeUpgrade         bool
	DesiredControlPlaneVersion string
	DesiredNodeVersion         string
	InPlaceUpgrade             bool
}

type clusterMigrator struct {
	projectID string
	cluster   *container.Cluster
	handler   operations.Handler
	clients   *pkg.Clients
	opts      *Options
	factory   func(n *container.NodePool) migrate.Migrator
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

// Migrate performs upgrade on the Cluster
func (m *clusterMigrator) Migrate(ctx context.Context) error {
	// implementation incoming in another pull.

	return nil
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
