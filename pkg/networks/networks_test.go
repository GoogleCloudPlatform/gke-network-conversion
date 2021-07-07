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
package networks

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"legacymigration/pkg"
	"legacymigration/pkg/migrate"
	"legacymigration/pkg/operations"
	"legacymigration/test"
)

var (
	testHandler = operations.NewHandler(1*time.Microsecond, 1*time.Millisecond)
)

func TestNetworkMigrator_Complete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	legacyNetwork := &compute.Network{
		Name:      test.SelectedNetwork,
		IPv4Range: "10.20.0.0/16",
	}

	cases := []struct {
		desc         string
		ctx          context.Context
		m            *networkMigrator
		wantChildren int
		wantErr      string
	}{
		{
			desc: "ListClusterError",
			ctx:  ctx,
			m: testNetworkMigrator(
				legacyNetwork,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).ListClustersErr = errors.New("list cluster err")
					return clients
				}(test.DefaultClients()),
			),
			wantErr: "list cluster err",
		},
		{
			desc: "Success",
			ctx:  ctx,
			m: testNetworkMigrator(
				legacyNetwork,
				test.DefaultClients(),
			),
			wantChildren: len(test.DefaultFakeContainer().ListClustersResp.Clusters),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.m.Complete(tc.ctx)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("networkMigrator.Complete diff (-want +got):\n%s", diff)
			}

			gotChildren := len(tc.m.children)
			if tc.wantChildren != gotChildren {
				t.Errorf("networkMigrator.Complete did not produce expected child migrators (want: %d, got: %d)", tc.wantChildren, gotChildren)
			}
		})
	}
}

func TestNetworkMigrator_Validate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	legacyNetwork := &compute.Network{
		Name:      test.SelectedNetwork,
		IPv4Range: "10.20.0.0/16",
	}

	cases := []struct {
		desc     string
		ctx      context.Context
		children []migrate.Migrator
		wantErr  string
	}{
		{
			desc: "Success",
			ctx:  ctx,
			children: []migrate.Migrator{
				&migrate.FakeMigrator{},
			},
		},
		{
			desc: "Children validation error",
			ctx:  ctx,
			children: []migrate.Migrator{
				&migrate.FakeMigrator{ValidateError: errors.New("validation error")},
			},
			wantErr: "validation error",
		},
		{
			desc: "Context canceled",
			ctx:  canceled,
			children: []migrate.Migrator{
				&migrate.FakeMigrator{},
			},
			wantErr: "context canceled",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			m := testNetworkMigrator(legacyNetwork, test.DefaultClients())
			m.children = tc.children

			err := m.Validate(tc.ctx)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("networkMigrator.Validate diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNetworkMigrator_Migrate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	clients := test.DefaultClients()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	legacyNetwork := &compute.Network{
		Name:      test.SelectedNetwork,
		IPv4Range: "10.20.0.0/16",
	}

	cases := []struct {
		desc    string
		ctx     context.Context
		m       *networkMigrator
		wantErr string
	}{
		{
			desc: "Success",
			ctx:  ctx,
			m:    testNetworkMigrator(legacyNetwork, clients),
		},
		{
			desc: "Missing zones",
			ctx:  ctx,
			m: testNetworkMigrator(
				legacyNetwork,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Container.(*test.FakeContainer).ListClustersResp = &container.ListClustersResponse{
						Clusters:     []*container.Cluster{&test.PrePatchCluster},
						MissingZones: []string{"zone-0-a", "zone-1-b"},
					}
					return clients
				}(test.DefaultClients()),
			),
		},
		{
			desc: "VPC Network",
			ctx:  ctx,
			m:    testNetworkMigrator(&compute.Network{Name: test.SelectedNetwork}, clients),
		},
		{
			desc: "SwitchToCustomMode error",
			ctx:  ctx,
			m: testNetworkMigrator(
				legacyNetwork,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Compute.(*test.FakeCompute).SwitchToCustomModeErr = errors.New("not allowlisted")
					clients.Compute.(*test.FakeCompute).GetGlobalOperationErr = errors.New("not found")
					return clients
				}(test.DefaultClients()),
			),
			wantErr: "not allowlisted",
		},
		{
			desc: "SwitchToCustomMode in progress",
			ctx:  ctx,
			m: testNetworkMigrator(
				legacyNetwork,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Compute.(*test.FakeCompute).SwitchToCustomModeErr = errors.New("operation: operation-abc-123 already in progress")
					return clients
				}(test.DefaultClients()),
			),
		},
		{
			desc: "SwitchToCustomMode fails",
			ctx:  ctx,
			m: testNetworkMigrator(
				legacyNetwork,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Compute.(*test.FakeCompute).WaitOperationResp = &compute.Operation{
						Name:   test.GenericOperationName,
						Status: test.OperationDone,
						Error: &compute.OperationError{
							Errors: []*compute.OperationErrorErrors{
								{Message: "switch to custom mode failed"},
							},
						},
					}
					return clients
				}(test.DefaultClients()),
			),
			wantErr: "switch to custom mode failed",
		},
		{
			desc: "WaitOperation error",
			ctx:  ctx,
			m: testNetworkMigrator(
				legacyNetwork,
				func(clients *pkg.Clients) *pkg.Clients {
					clients.Compute.(*test.FakeCompute).WaitOperationErr = errors.New("wait error")
					return clients
				}(test.DefaultClients()),
			),
			wantErr: "wait error",
		},
		{
			desc:    "Context cancelled",
			ctx:     cancelled,
			m:       testNetworkMigrator(legacyNetwork, clients),
			wantErr: "context error: context canceled",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.m.Migrate(tc.ctx)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("networkMigrator.Migrate diff (-want +got):\n%s", diff)
			}
		})
	}
}

func testNetworkMigrator(n *compute.Network, c *pkg.Clients) *networkMigrator {
	return &networkMigrator{
		projectID:          test.ProjectName,
		network:            n,
		handler:            testHandler,
		clients:            c,
		concurrentClusters: 1,
		factory: func(c *container.Cluster) migrate.Migrator {
			return &migrate.FakeMigrator{}
		},
	}
}
