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

func TestNetworkMigrator_Migrate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	clients := test.DefaultClients()

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
			m:    testMigrator(legacyNetwork, clients),
		},
		// further tests incoming in another pull.
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

func testMigrator(n *compute.Network, c *pkg.Clients) *networkMigrator {
	return &networkMigrator{
		projectID:          test.ProjectName,
		network:            n,
		handler:            testHandler,
		clients:            c,
		concurrentClusters: 1,
		factory:            func(c *container.Cluster) migrate.Migrator { return &test.FakeMigrator{} },
	}
}
