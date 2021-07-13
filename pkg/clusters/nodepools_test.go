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

func TestNodePoolMigrator_Complete(t *testing.T) {
	cases := []struct {
		desc      string
		ctx       context.Context
		npDesired string
		cpVersion string
		want      string
		wantErr   string
	}{
		{
			desc:      "Success - resolve literal version",
			ctx:       context.Background(),
			npDesired: "1.19.10-gke.1700",
			want:      "1.19.10-gke.1700",
		},
		{
			desc:      "Success - resolve default alias",
			ctx:       context.Background(),
			npDesired: "-",
			cpVersion: "1.19.10-gke.1600",
			want:      "1.19.10-gke.1600",
		},
		{
			desc:      "Success - resolve minor version alias",
			ctx:       context.Background(),
			npDesired: "1.19",
			want:      "1.19.11-gke.1700",
		},
		{
			desc:      "Fail - cannot resolve",
			ctx:       context.Background(),
			npDesired: "1.22",
			wantErr:   "could not be resolved",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			m := testNodePoolMigrator()
			m.opts.DesiredNodeVersion = tc.npDesired
			m.resolvedDesiredControlPlaneVersion = tc.cpVersion

			err := m.Complete(tc.ctx)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("nodePoolMigrator.Complete diff (-want +got):\n%s", diff)
			}

			got := m.resolvedDesiredNodeVersion
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolvedDesiredNodeVersion diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNodePoolMigrator_Validate(t *testing.T) {
	cases := []struct {
		desc       string
		ctx        context.Context
		npDesired  string
		npResolved string
		npCurrent  string
		cpVersion  string
		want       string
		wantErr    string
	}{
		{
			desc:       "Success - minor version upgrade",
			ctx:        context.Background(),
			npDesired:  "1.19.11-gke.1700",
			npResolved: "1.19.11-gke.1700",
			npCurrent:  "1.19.10-gke.1700",
			cpVersion:  "1.19.10-gke.1700",
		},
		{
			desc:       "Fail - in-place node upgrade",
			ctx:        context.Background(),
			npDesired:  "1.19.10-gke.1700",
			npResolved: "1.19.10-gke.1700",
			npCurrent:  "1.19.10-gke.1700",
			wantErr:    "must be newer than current version",
		},
		{
			desc:       "Fail - outside version skew",
			ctx:        context.Background(),
			npDesired:  "1.19.10-gke.1700",
			npResolved: "1.19.10-gke.1700",
			npCurrent:  "1.17.17-gke.9100",
			cpVersion:  "1.17.17-gke.9100",
			wantErr:    "must be within 1 minor versions of desired control plane version",
		},
	}
	for _, tc := range cases {
		m := testNodePoolMigrator()
		m.opts.DesiredNodeVersion = tc.npDesired
		m.nodePool.Version = tc.npCurrent
		m.resolvedDesiredControlPlaneVersion = tc.cpVersion
		m.resolvedDesiredNodeVersion = tc.npResolved

		t.Run(tc.desc, func(t *testing.T) {
			err := m.Validate(tc.ctx)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("nodePoolMigrator.Validate diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNodePoolMigrator_isUpgradeRequired(t *testing.T) {
	cases := []struct {
		desc    string
		URLs    []string
		want    bool
		wantErr string
	}{
		{
			desc: "Single InstanceGroupManager",
			URLs: []string{
				test.InstanceGroupManagerZoneA0,
			},
			want: true,
		},
		{
			desc: "Multiple InstanceGroupManagers",
			URLs: []string{
				test.InstanceGroupManagerZoneA0,
				test.InstanceGroupManagerZoneA1,
			},
			want: true,
		},
		{
			desc: "Regional InstanceGroupManager",
			URLs: []string{
				test.InstanceGroupManagerRegionA,
			},
			want: true,
		},
		{
			// No underlying template to update via an upgrade.
			desc: "No InstanceGroupManagers",
		},
		{
			// InstanceGroups do not have a NodeTemplate and should not appear in a InstanceGroup NodePool's list of URLs.
			desc: "Unhandled InstanceGroups",
			URLs: []string{
				fmt.Sprintf("%s/projects/%s/zones/%s/instanceGroups/%s", test.ComputeAPI, test.ProjectName, test.ZoneA0, "instanceGroup0"),
			},
			wantErr: "error(s) encountered obtaining an InstanceTemplate for NodePool",
		},
		{
			desc: "Matching InstanceGroup",
			URLs: []string{
				test.InstanceGroupManagerZoneA1,
			},
			want: true,
		},
		{
			desc: "One matching InstanceGroup",
			URLs: []string{
				fmt.Sprintf("%s/projects/%s/zones/%s/instanceGroups/%s", test.ComputeAPI, test.ProjectName, test.ZoneA0, "instanceGroup0"),
				test.InstanceGroupManagerZoneA1,
			},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			m := testNodePoolMigrator()
			m.nodePool.InstanceGroupUrls = tc.URLs

			got, err := m.isUpgradeRequired(context.Background())
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("nodePoolMigrator.isUpgradeRequired error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("nodePoolMigrator.isUpgradeRequired diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNodePoolMigrator_Migrate(t *testing.T) {
	cases := []struct {
		desc    string
		clients *pkg.Clients
		wantErr string
		wantLog string
	}{
		{
			desc:    "Migrate node pool",
			clients: test.DefaultClients(),
			wantLog: "NodePool projects/test-project/locations/region-a/operations/operation-update-nodepool upgraded",
		},
		{
			desc: "UpdateNodePool error",
			clients: func(clients *pkg.Clients) *pkg.Clients {
				clients.Container.(*test.FakeContainer).UpdateNodePoolErr = errors.New("unrecoverable error")
				clients.Container.(*test.FakeContainer).GetOperationErr = errors.New("not found")
				return clients
			}(test.DefaultClients()),
			wantErr: "error upgrading NodePool",
		},
		{
			desc: "NodePool upgrade in progress",
			clients: func(clients *pkg.Clients) *pkg.Clients {
				clients.Container.(*test.FakeContainer).UpdateNodePoolErr = errors.New("operation: operation-abc-123 already in progress")
				return clients
			}(test.DefaultClients()),
			wantLog: "upgraded",
		},
		{
			desc: "Polling failure during UpdateNodePool operation",
			clients: func(clients *pkg.Clients) *pkg.Clients {
				clients.Container.(*test.FakeContainer).GetOperationErr = errors.New("operation get failed")
				return clients
			}(test.DefaultClients()),
			wantErr: "error retrieving Operation projects/test-project/locations/region-a/operations/operation-update-nodepool: operation get failed",
		},
		{
			desc: "UpdateNodePool operation failure",
			clients: func(clients *pkg.Clients) *pkg.Clients {
				clients.Container.(*test.FakeContainer).GetOperationResp = &container.Operation{
					Name:   "op",
					Status: test.OperationDone,
					Error:  &container.Status{Message: "operation failed"},
				}
				return clients
			}(test.DefaultClients()),
			wantErr: "error waiting on Operation projects/test-project/locations/region-a/operations/operation-update-nodepool: operation failed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			m := testNodePoolMigrator()
			m.clients = tc.clients
			buf := &bytes.Buffer{}
			log.StandardLogger().SetOutput(buf)

			err := m.Migrate(context.Background())
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("nodePoolMigrator.Migrate diff (-want +got):\n%s", diff)
			}
			if diff := !strings.Contains(buf.String(), tc.wantLog); tc.wantLog != "" && diff {
				t.Errorf("nodePoolMigrator.Migrate missing log output:\n\twanted entry: %s\n\tgot entries: %s", tc.wantLog, buf.String())
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

func testNodePoolMigrator() *nodePoolMigrator {
	return &nodePoolMigrator{
		clusterMigrator: &clusterMigrator{
			projectID: test.ProjectName,
			cluster: &container.Cluster{
				Name:     test.ClusterName,
				Location: test.RegionA,
				Network:  test.SelectedNetwork,
				ReleaseChannel: &container.ReleaseChannel{
					Channel: "UNSPECIFIED",
				},
			},
			opts:                               &Options{},
			handler:                            testHandler,
			clients:                            test.DefaultClients(),
			serverConfig:                       ServerConfig,
			resolvedDesiredControlPlaneVersion: "",
		},
		nodePool: &container.NodePool{
			Name: "pool",
		},
		upgradeRequired: true,
	}
}
