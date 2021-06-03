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
		// further tests incoming in another pull.
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
