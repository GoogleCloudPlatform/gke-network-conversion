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
	// implementation incoming in another pull.

	return nil
}

type ComputeOperation struct {
	ProjectID string
	Operation *compute.Operation
	Client    pkg.ComputeService
}

// String returns the resource path of the operation.
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

// IsFinished returns whether the operation is complete or the operation error status.
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
