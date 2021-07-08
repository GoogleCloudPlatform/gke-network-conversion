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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
		DesiredControlPlaneVersion: pkg.DefaultVersion,
		InPlaceControlPlaneUpgrade: false,
	}
)

func TestClusterMigrator_Complete(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc         string
		clients      *pkg.Clients
		opts         *Options
		want         string
		wantChildren int
		wantErr      string
	}{
		{
			desc:         "Success",
			clients:      test.DefaultClients(),
			opts:         testOptions,
			want:         "1.19.10-gke.1600",
			wantChildren: len(test.DefaultFakeContainer().ListNodePoolsResp.NodePools),
		},
		{
			desc: "ListNodePools error",
			clients: func(clients *pkg.Clients) *pkg.Clients {
				clients.Container.(*test.FakeContainer).ListNodePoolsErr = errors.New("an error")
				return clients
			}(test.DefaultClients()),
			opts:    testOptions,
			wantErr: "error retrieving NodePools for Cluster",
		},
		{
			desc: "GetServerConfig error",
			clients: func(clients *pkg.Clients) *pkg.Clients {
				clients.Container.(*test.FakeContainer).GetServerConfigErr = errors.New("an error")
				return clients
			}(test.DefaultClients()),
			opts:    testOptions,
			wantErr: "error retrieving ServerConfig for Cluster",
		},
		{
			desc:    "Success - in-place upgrade",
			clients: test.DefaultClients(),
			opts: &Options{
				ConcurrentNodePools:        1,
				InPlaceControlPlaneUpgrade: true,
			},
			wantChildren: len(test.DefaultFakeContainer().ListNodePoolsResp.NodePools),
			want:         "1.19.10-gke.1600",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			m := testClusterMigrator(&test.PrePatchCluster, testOptions, tc.clients)

			err := m.Complete(context.Background())
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("clusterMigrator.Complete diff (-want +got):\n%s", diff)
			}

			got := m.resolvedDesiredControlPlaneVersion
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolvedDesiredControlPlaneVersion diff (-want +got):\n%s", diff)
			}

			gotChildren := len(m.children)
			if tc.wantChildren != gotChildren {
				t.Errorf("clusterMigrator.Complete did not produce expected child migrators (want: %d, got: %d)", tc.wantChildren, gotChildren)
			}
		})
	}
}

func TestClusterMigrator_Complete_Error(t *testing.T) {
	want := "child error"
	m := testClusterMigrator(&test.PrePatchCluster, testOptions, test.DefaultClients())
	m.factory = func(_ *container.NodePool) migrate.Migrator {
		return &migrate.FakeMigrator{CompleteError: errors.New(want)}
	}

	err := m.Complete(context.Background())
	if diff := test.ErrorDiff(want, err); diff != "" {
		t.Errorf("clusterMigrator.Complete diff (-want +got):\n%s", diff)
	}
}

func TestClusterMigrator_Validate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc     string
		resolved string
		config   *container.ServerConfig
		children []migrate.Migrator
		wantErr  string
	}{
		{
			desc:     "Success - no children",
			resolved: "1.19.10-gke.1700",
			config:   ServerConfig,
			children: []migrate.Migrator{},
		},
		{
			desc:     "Success - single child",
			resolved: "1.19.10-gke.1700",
			config:   ServerConfig,
			children: []migrate.Migrator{
				&migrate.FakeMigrator{},
			},
		},
		{
			desc:     "Success - multiple child",
			resolved: "1.19.10-gke.1700",
			config:   ServerConfig,
			children: []migrate.Migrator{
				&migrate.FakeMigrator{},
				&migrate.FakeMigrator{},
			},
		},
		{
			desc:     "Fail - not an upgrade",
			resolved: "1.18.19-gke.1700",
			config:   ServerConfig,
			wantErr:  "must be newer than current version",
		},
		{
			desc:     "Fail - Child migrator failure",
			resolved: "1.19.10-gke.1700",
			config:   ServerConfig,
			children: []migrate.Migrator{
				&migrate.FakeMigrator{ValidateError: errors.New("child error")},
			},
			wantErr: "child error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			m := testClusterMigrator(&test.PrePatchCluster, testOptions, test.DefaultClients())
			m.resolvedDesiredControlPlaneVersion = tc.resolved
			m.serverConfig = tc.config
			m.children = tc.children

			err := m.Validate(context.Background())
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("clusterMigrator.Validate diff (-want +got):\n%s", diff)
			}
		})
	}
}

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
			m:    testClusterMigrator(&c, testOptions, clients),
		},
		{
			desc: "Subnet field already present",
			ctx:  ctx,
			m:    testClusterMigrator(&container.Cluster{Subnetwork: "subnet"}, testOptions, clients),
		},
		{
			desc: "UpdateMaster in progress",
			ctx:  ctx,
			m: testClusterMigrator(
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
			m: testClusterMigrator(
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
			m: testClusterMigrator(
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
			m: testClusterMigrator(
				&c,
				testOptions,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).GetClusterResp = &test.PrePatchCluster
					return clients
				}(test.DefaultClients())),
			wantErr: "subnetwork field is empty for cluster",
		},
		{
			desc: "Polling failure",
			ctx:  ctx,
			m: testClusterMigrator(
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
			m: testClusterMigrator(
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
			m:       testClusterMigrator(&c, testOptions, clients),
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

func testClusterMigrator(c *container.Cluster, opts *Options, clients *pkg.Clients) *clusterMigrator {
	return &clusterMigrator{
		projectID: test.ProjectName,
		cluster:   c,
		handler:   testHandler,
		clients:   clients,
		opts:      opts,
		factory: func(_ *container.NodePool) migrate.Migrator {
			return &migrate.FakeMigrator{}
		},
	}
}
