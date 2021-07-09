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
package cmd

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"legacymigration/pkg"
	"legacymigration/pkg/clusters"
	"legacymigration/pkg/migrate"
	"legacymigration/test"
)

func TestMigrateOptions_ValidateFlagsNoProject(t *testing.T) {
	opts := migrateOptions{}
	want := "--project not provided or empty"

	got := opts.ValidateFlags()

	if diff := test.ErrorDiff(want, got); diff != "" {
		t.Errorf("MigrateOptions.ValidateFlags diff (-want +got):\n%s", diff)
	}
}

func TestMigrateOptions_ValidateFlags(t *testing.T) {
	cases := []struct {
		desc string
		opts migrateOptions
		want string
	}{
		{
			desc: "Minimum required options",
			opts: defaultOptions(),
		},
		{
			desc: "Empty options",
			opts: migrateOptions{},
			want: "--project not provided or empty",
		},
		{
			desc: "concurrentClusters too low",
			opts: func(o migrateOptions) migrateOptions {
				o.concurrentClusters = 0
				return o
			}(defaultOptions()),
			want: "--concurrent-clusters must be an integer greater than 0",
		},
		{
			desc: "polling too low",
			opts: func(o migrateOptions) migrateOptions {
				o.pollingInterval = 1 * time.Second
				return o
			}(defaultOptions()),
			want: "--polling-interval must greater than or equal to 10 seconds",
		},
		{
			desc: "pollingDeadline too low",
			opts: func(o migrateOptions) migrateOptions {
				o.pollingDeadline = 1 * time.Second
				return o
			}(defaultOptions()),
			want: "--polling-deadline must greater than or equal to 5 minutes",
		},
		{
			desc: "polling greater than pollingDeadline",
			opts: func(o migrateOptions) migrateOptions {
				o.pollingInterval = 1 * time.Hour
				return o
			}(defaultOptions()),
			want: "--polling-deadline=20m0s must be greater than --polling-interval=1h0m0s",
		},
		{
			desc: "In place upgrade and desired version",
			opts: func(o migrateOptions) migrateOptions {
				o.inPlaceControlPlaneUpgrade = true
				o.desiredControlPlaneVersion = "1.19"
				return o
			}(defaultOptions()),
			want: "specify --in-place-control-plane or provide a version for --control-plane-version",
		},
		{
			desc: "In place upgrade",
			opts: func(o migrateOptions) migrateOptions {
				o.inPlaceControlPlaneUpgrade = true
				o.desiredControlPlaneVersion = ""
				o.desiredNodeVersion = clusters.DefaultVersion
				return o
			}(defaultOptions()),
		},
		{
			desc: "Both versions set to latest",
			opts: func(o migrateOptions) migrateOptions {
				o.desiredControlPlaneVersion = clusters.LatestVersion
				o.desiredNodeVersion = clusters.LatestVersion
				return o
			}(defaultOptions()),
		},
		{
			desc: "minor aliases provided",
			opts: func(o migrateOptions) migrateOptions {
				o.desiredControlPlaneVersion = "1.19"
				o.desiredNodeVersion = "1.19"
				return o
			}(defaultOptions()),
		},
		{
			desc: "Control plane minor aliases provided",
			opts: func(o migrateOptions) migrateOptions {
				o.desiredControlPlaneVersion = "1.19"
				o.desiredNodeVersion = clusters.DefaultVersion
				return o
			}(defaultOptions()),
		},
		{
			desc: "Outside version skew",
			opts: func(o migrateOptions) migrateOptions {
				o.desiredControlPlaneVersion = "1.19"
				o.desiredNodeVersion = "1.17"
				return o
			}(defaultOptions()),
			want: "must be within 1 minor versions of desired control plane version",
		},
		{
			desc: "Invalid control plane format",
			opts: func(o migrateOptions) migrateOptions {
				o.desiredControlPlaneVersion = "x.y"
				return o
			}(defaultOptions()),
			want: `--control-plane-version="x.y" is not valid`,
		},
		{
			desc: "Invalid node format",
			opts: func(o migrateOptions) migrateOptions {
				o.desiredNodeVersion = "x.y"
				o.desiredNodeVersion = "x.y"
				return o
			}(defaultOptions()),
			want: `--node-version="x.y" is not valid`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.opts.ValidateFlags()
			if diff := test.ErrorDiff(tc.want, got); diff != "" {
				t.Errorf("MigrateOptions.ValidateFlags diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMigrateOptions_Complete(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		desc    string
		opts    migrateOptions
		wantErr string
	}{
		{
			desc: "Success",
			opts: defaultOptions(),
		},
		{
			desc: "Network miss",
			opts: func(o migrateOptions) migrateOptions {
				o.selectedNetwork = "miss"
				return o
			}(defaultOptions()),
			wantErr: "unable to find network",
		},
		{
			desc: "ListNetworks error",
			opts: func(o migrateOptions) migrateOptions {
				o.fetchClientFunc = func(ctx context.Context, basePath string, authedClient *http.Client) (*pkg.Clients, error) {
					clients := test.DefaultClients()
					clients.Compute.(*test.FakeCompute).ListNetworksErr = errors.New("ListNetworks error")
					return clients, nil
				}
				return o
			}(defaultOptions()),
			wantErr: "error listing networks: ListNetworks error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.opts.Complete(ctx)
			if diff := test.ErrorDiff(tc.wantErr, got); diff != "" {
				t.Errorf("MigrateOptions.Complete diff (-want +got):\n%s", diff)
			}

			if tc.opts.clients == nil {
				t.Fatalf("opts.Clients is nil")
			}
			if tc.opts.clients.Compute == nil {
				t.Errorf("Compute client is nil")
			}
			if tc.opts.clients.Container == nil {
				t.Errorf("Container client is nil")
			}
		})
	}
}

func TestMigrateOptions_Run(t *testing.T) {
	cases := []struct {
		desc    string
		opts    migrateOptions
		wantErr string
		wantLog string
	}{
		{
			desc:    "Empty migrator list",
			opts:    migrateOptions{},
			wantLog: "Initiate resource conversion",
		},
		{
			opts: migrateOptions{
				validateOnly: true,
			},
			wantLog: "skipping conversion",
		},
		{
			opts: migrateOptions{
				migrators: []migrate.Migrator{
					&migrate.FakeMigrator{},
				},
			},
			wantLog: "Initiate resource conversion.",
		},
		{
			opts: migrateOptions{
				migrators: []migrate.Migrator{
					&migrate.FakeMigrator{
						MigrateError: errors.New("migrate error"),
					},
				},
			},
			wantLog: "Initiate resource conversion",
			wantErr: "migrate error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log.StandardLogger().SetOutput(buf)

			got := tc.opts.Run(context.Background())
			if diff := test.ErrorDiff(tc.wantErr, got); diff != "" {
				t.Errorf("migrateOptions.Run diff (-want +got):\n%s", diff)
			}

			if diff := !strings.Contains(buf.String(), tc.wantLog); tc.wantLog != "" && diff {
				t.Errorf("migrateOptions.Run missing log output:\n\twanted entry: %s\n\tgot entries: %s", tc.wantLog, buf.String())
			}
		})
	}
}

func testClientFunc(_ context.Context, _ string, _ *http.Client) (*pkg.Clients, error) {
	return test.DefaultClients(), nil
}

func defaultOptions() migrateOptions {
	return migrateOptions{
		projectID:                  test.ProjectName,
		selectedNetwork:            test.SelectedNetwork,
		desiredControlPlaneVersion: clusters.DefaultVersion,
		desiredNodeVersion:         clusters.DefaultVersion,
		concurrentClusters:         1,
		pollingInterval:            10 * time.Minute,
		pollingDeadline:            20 * time.Minute,
		fetchClientFunc:            testClientFunc,
	}
}
