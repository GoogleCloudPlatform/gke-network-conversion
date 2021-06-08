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
	"errors"
	"testing"
	"time"

	"google.golang.org/api/container/v1"
	"legacymigration/pkg"
	"legacymigration/pkg/migrate"
	"legacymigration/pkg/operations"
	"legacymigration/test"
)

var (
	testHandler = operations.NewHandler(1*time.Microsecond, 1*time.Millisecond)
	testOptions = &Options{
		ConcurrentNodePools:        1,
		WaitForNodeUpgrade:         true,
		DesiredControlPlaneVersion: pkg.DefaultVersion,
		DesiredNodeVersion:         pkg.DefaultVersion,
		InPlaceUpgrade:             false,
	}
)

func TestClusterMigrator_Migrate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	clients := test.DefaultClients()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	c := test.PrePatchCluster

	cases := []struct {
		desc    string
		ctx     context.Context
		m       *clusterMigrator
		wantErr string
	}{
		{
			desc: "Success",
			ctx:  ctx,
			m:    testMigrator(&c, testOptions, clients),
		},
		{
			desc: "Subnet field already present",
			ctx:  ctx,
			m:    testMigrator(&container.Cluster{Subnetwork: "subnet"}, testOptions, clients),
		},
		{
			desc: "UpdateMaster in progress",
			ctx:  ctx,
			m: testMigrator(
				&c,
				testOptions,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).UpdateMasterErr = errors.New("operation projects/test-project/locations/region-a/operations/operation-update-master already in progress")
					return clients
				}(test.DefaultClients())),
		},
		{
			desc: "UpdateMaster error",
			ctx:  ctx,
			m: testMigrator(
				&c,
				testOptions,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).UpdateMasterErr = errors.New("unrecoverable error")
					clients.Container.(*test.FakeContainer).GetOperationErr = errors.New("not found")
					return clients
				}(test.DefaultClients())),
			wantErr: "error upgrading control plane for Cluster",
		},
		{
			desc: "GetCluster error",
			ctx:  ctx,
			m: testMigrator(
				&c,
				testOptions,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).GetClusterErr = errors.New("cannot get cluster")
					return clients
				}(test.DefaultClients())),
			wantErr: "unable to confirm subnetwork value for cluster",
		},
		{
			desc: "Patch not performed",
			ctx:  ctx,
			m: testMigrator(
				&c,
				testOptions,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).GetClusterResp = &test.PrePatchCluster
					return clients
				}(test.DefaultClients())),
			wantErr: "subnetwork field is empty for cluster",
		},
		{
			desc: "ListNodePools error",
			ctx:  ctx,
			m: testMigrator(
				&c,
				testOptions,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).ListNodePoolsErr = errors.New("cannot list nodePools")
					return clients
				}(test.DefaultClients())),
			wantErr: "error retrieving NodePools",
		},
		{
			desc: "Polling failure",
			ctx:  ctx,
			m: testMigrator(
				&c,
				testOptions,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).GetOperationErr = errors.New("operation get failed")
					return clients
				}(test.DefaultClients())),
			wantErr: "error retrieving Operation projects/test-project/locations/region-a/operations/operation-update-master: operation get failed",
		},
		{
			desc: "Operation failure",
			ctx:  ctx,
			m: testMigrator(
				&c,
				testOptions,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).GetOperationResp = &container.Operation{
						Name:   "op",
						Status: test.OperationDone,
						Error:  &container.Status{Message: "operation failed"},
					}
					return clients
				}(test.DefaultClients())),
			wantErr: "error waiting on Operation projects/test-project/locations/region-a/operations/operation-update-master: operation failed",
		},
		{
			desc:    "Context cancelled",
			ctx:     cancelled,
			m:       testMigrator(&c, testOptions, clients),
			wantErr: "context error: context canceled",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.m.Migrate(tc.ctx)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("clusterMigrator.Migrate diff (-want +got):\n%s", diff)
			}
		})
	}
}

func testMigrator(c *container.Cluster, opts *Options, clients *pkg.Clients) *clusterMigrator {
	return &clusterMigrator{
		projectID: test.ProjectName,
		cluster:   c,
		handler:   testHandler,
		clients:   clients,
		opts:      opts,
		factory:   func(_ *container.NodePool) migrate.Migrator { return &test.FakeMigrator{} },
	}
}
