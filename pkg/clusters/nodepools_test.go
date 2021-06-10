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
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/container/v1"
	"legacymigration/pkg"
	"legacymigration/test"
)

func TestNodePoolMigrator_Migrate(t *testing.T) {
	ctx := context.Background()
	clients := test.DefaultClients()
	cm := testMigrator(&test.PrePatchCluster, testOptions, clients)

	cases := []struct {
		desc    string
		ctx     context.Context
		m       *nodePoolMigrator
		wantErr string
		wantLog string
	}{
		{
			desc: "Single InstanceGroupManager",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: cm,
				nodePool: &container.NodePool{
					Name:      "np",
					Locations: []string{test.ZoneA0},
					InstanceGroupUrls: []string{
						test.InstanceGroupManagerZoneA0,
					},
				},
			},
			wantLog: "Upgrading NodePool",
		},
		{
			desc: "Multiple InstanceGroupManagers",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: cm,
				nodePool: &container.NodePool{
					Name:      "np",
					Locations: []string{test.ZoneA0},
					InstanceGroupUrls: []string{
						test.InstanceGroupManagerZoneA0,
						test.InstanceGroupManagerZoneA1,
					},
				},
			},
			wantLog: "Upgrading NodePool",
		},
		{
			desc: "Regional InstanceGroupManager",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: cm,
				nodePool: &container.NodePool{
					Name:      "np",
					Locations: []string{test.ZoneA0},
					InstanceGroupUrls: []string{
						test.InstanceGroupManagerRegionA,
					},
				},
			},
			wantLog: "Upgrading NodePool",
		},
		{
			// No underlying template to update via an upgrade.
			desc: "No InstanceGroupManagers",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: cm,
				nodePool: &container.NodePool{
					Name:      "np",
					Locations: []string{test.ZoneA0},
				},
			},
			wantLog: "Upgrade not required for NodePool",
		},
		{
			// InstanceGroups do not have a NodeTemplate and should not appear in a InstanceGroup NodePool's list of URLs.
			desc: "Unhandled InstanceGroups",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: cm,
				nodePool: &container.NodePool{
					Name:      "np",
					Locations: []string{test.ZoneA0},
					InstanceGroupUrls: []string{
						fmt.Sprintf("%s/projects/%s/zones/%s/instanceGroups/%s", test.ComputeAPI, test.ProjectName, test.ZoneA0, "instanceGroup0"),
					},
				},
			},
			wantErr: "error(s) encountered obtaining an InstanceTemplate for NodePool",
		},
		{
			desc: "matching InstanceGroup",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: cm,
				nodePool: &container.NodePool{
					Name:      "np",
					Locations: []string{test.ZoneA0},
					InstanceGroupUrls: []string{
						test.InstanceGroupManagerZoneA1,
					},
				},
			},
			wantLog: "Upgrading NodePool",
		},
		{
			desc: "One matching InstanceGroup",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: cm,
				nodePool: &container.NodePool{
					Name:      "np",
					Locations: []string{test.ZoneA0},
					InstanceGroupUrls: []string{
						fmt.Sprintf("%s/projects/%s/zones/%s/instanceGroups/%s", test.ComputeAPI, test.ProjectName, test.ZoneA0, "instanceGroup0"),
						test.InstanceGroupManagerZoneA1,
					},
				},
			},
			wantLog: "Upgrading NodePool",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log.StandardLogger().SetOutput(buf)

			err := tc.m.Migrate(tc.ctx)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("nodePoolMigrator.Migrate diff (-want +got):\n%s", diff)
			}

			if diff := !strings.Contains(buf.String(), tc.wantLog); tc.wantLog != "" && diff {
				t.Errorf("nodePoolMigrator.Migrate missing log output:\n\twanted entry: %s\n\tgot entries: %s", tc.wantLog, buf.String())
			}
		})
	}
}

func TestNodePoolMigrator_migrate(t *testing.T) {
	ctx := context.Background()
	clients := test.DefaultClients()

	cases := []struct {
		desc    string
		ctx     context.Context
		m       *nodePoolMigrator
		wantErr string
		wantLog string
	}{
		{
			desc: "Do not wait for upgrade",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: testMigrator(
					&test.PrePatchCluster,
					&Options{
						ConcurrentNodePools:        1,
						WaitForNodeUpgrade:         false,
						DesiredControlPlaneVersion: pkg.DefaultVersion,
						DesiredNodeVersion:         pkg.DefaultVersion,
						InPlaceUpgrade:             false,
					},
					clients),
				nodePool: &container.NodePool{
					Name: "np",
				},
			},
			wantLog: "Not waiting on upgrade for NodePool",
		},
		{
			desc: "UpdateNodePool error",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: testMigrator(
					&test.PrePatchCluster,
					testOptions,
					func(clients *pkg.Clients) *pkg.Clients {
						clients.Container.(*test.FakeContainer).UpdateNodePoolErr = errors.New("unrecoverable error")
						clients.Container.(*test.FakeContainer).GetOperationErr = errors.New("not found")
						return clients
					}(test.DefaultClients())),
				nodePool: &container.NodePool{
					Name: "np",
				},
			},
			wantErr: "error upgrading NodePool",
		},
		{
			desc: "NodePool upgrade in progress",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: testMigrator(
					&test.PrePatchCluster,
					testOptions,
					func(clients *pkg.Clients) *pkg.Clients {
						clients.Container.(*test.FakeContainer).UpdateNodePoolErr = errors.New("operation: operation-abc-123 already in progress")
						return clients
					}(test.DefaultClients())),
				nodePool: &container.NodePool{
					Name: "np",
				},
			},
			wantLog: "upgraded",
		},
		{
			desc: "Not waiting on NodePool upgrade in progress",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: testMigrator(
					&test.PrePatchCluster,
					&Options{
						ConcurrentNodePools:        1,
						WaitForNodeUpgrade:         false,
						DesiredControlPlaneVersion: pkg.DefaultVersion,
						DesiredNodeVersion:         pkg.DefaultVersion,
						InPlaceUpgrade:             false,
					},
					func(clients *pkg.Clients) *pkg.Clients {
						clients.Container.(*test.FakeContainer).UpdateNodePoolErr = errors.New("operation: operation-abc-123 already in progress")
						return clients
					}(test.DefaultClients())),
				nodePool: &container.NodePool{
					Name: "np",
				},
			},
			wantLog: "Not waiting on upgrade for NodePool",
		},
		{
			desc: "In-place upgrade",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: testMigrator(
					&test.PrePatchCluster,
					&Options{
						ConcurrentNodePools:        1,
						WaitForNodeUpgrade:         true,
						DesiredControlPlaneVersion: pkg.DefaultVersion,
						DesiredNodeVersion:         pkg.DefaultVersion,
						InPlaceUpgrade:             true,
					},
					clients),
				nodePool: &container.NodePool{
					Name:    "np",
					Version: "1.20.1",
				},
			},
			wantLog: "to version \\\"1.20.1\\\"",
		},
		{
			desc: "Polling failure during UpdateNodePool operation",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: testMigrator(
					&test.PrePatchCluster,
					testOptions,
					func(clients *pkg.Clients) *pkg.Clients {
						clients.Container.(*test.FakeContainer).GetOperationErr = errors.New("operation get failed")
						return clients
					}(test.DefaultClients())),
				nodePool: &container.NodePool{
					Name: "np",
				},
			},
			wantErr: "error retrieving Operation projects/test-project/locations/region-a/operations/operation-update-nodepool: operation get failed",
		},
		{
			desc: "UpdateNodePool operation failure",
			ctx:  ctx,
			m: &nodePoolMigrator{
				clusterMigrator: testMigrator(
					&test.PrePatchCluster,
					testOptions,
					func(clients *pkg.Clients) *pkg.Clients {
						clients.Container.(*test.FakeContainer).GetOperationResp = &container.Operation{
							Name:   "op",
							Status: test.OperationDone,
							Error:  &container.Status{Message: "operation failed"},
						}
						return clients
					}(test.DefaultClients())),
				nodePool: &container.NodePool{
					Name: "np",
				},
			},
			wantErr: "error waiting on Operation projects/test-project/locations/region-a/operations/operation-update-nodepool: operation failed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log.StandardLogger().SetOutput(buf)

			err := tc.m.migrate(tc.ctx)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("nodePoolMigrator.Migrate diff (-want +got):\n%s", diff)
			}

			if diff := !strings.Contains(buf.String(), tc.wantLog); tc.wantLog != "" && diff {
				t.Errorf("nodePoolMigrator.migrate missing log output:\n\twanted entry: %s\n\tgot entries: %s", tc.wantLog, buf.String())
			}
		})
	}
}

func TestGetName(t *testing.T) {
	cases := []struct {
		desc string
		path string
		want string
	}{
		{
			desc: "fullpath",
			path: "projects/x/locations/y/resources/z",
			want: "z",
		},
		{
			desc: "url",
			path: "https://container.googleapis.com/container/v1/projects/x/locations/y/resources/z",
			want: "z",
		},
		{
			desc: "no slashes",
			path: "x",
			want: "x",
		},
		{
			desc: "single slash",
			path: "z/x",
			want: "x",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := getName(tc.path)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("getName diff (-want +got):\n%s", diff)
			}
		})
	}
}
