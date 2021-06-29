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
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/container/v1"
	"legacymigration/test"
)

const (
	NoChannel = ""
	Rapid     = "Rapid"
	Regular   = "Regular"
	Stable    = "Stable"
)

var (
	ServerConfig = &container.ServerConfig{
		Channels: []*container.ReleaseChannelConfig{
			{
				Channel:        Rapid,
				DefaultVersion: "1.20.6-gke.1400",
				ValidVersions: []string{
					"1.21.1-gke.1800",
					"1.20.7-gke.1800",
					"1.20.6-gke.1400",
				},
			}, {
				Channel:        Regular,
				DefaultVersion: "1.19.10-gke.1600",
				ValidVersions: []string{
					"1.20.6-gke.1000",
					"1.19.10-gke.1700",
					"1.19.10-gke.1600",
				},
			}, {
				Channel:        Stable,
				DefaultVersion: "1.18.17-gke.1901",
				ValidVersions: []string{
					"1.19.10-gke.1000",
					"1.18.18-gke.1100",
					"1.18.17-gke.1901",
				},
			},
		},
		DefaultClusterVersion: "1.19.10-gke.1600",
		ValidMasterVersions: []string{
			"1.20.7-gke.1800",
			"1.20.6-gke.1000",
			"1.19.11-gke.1700",
			"1.19.10-gke.1700",
			"1.18.19-gke.1700",
			"1.18.18-gke.1700",
			"1.17.17-gke.9100",
			"1.17.17-gke.8200",
		},
		ValidNodeVersions: []string{
			"1.20.7-gke.1800",
			"1.20.6-gke.1000",
			"1.19.11-gke.1700",
			"1.19.10-gke.1700",
			"1.18.19-gke.1700",
			"1.18.18-gke.1700",
			"1.17.17-gke.9100",
			"1.17.17-gke.8200",
		},
	}
)

func TestGetVersions(t *testing.T) {
	cases := []struct {
		desc         string
		ServerConfig *container.ServerConfig
		channel      string
		resource     Resource
		wantDefault  string
		wantVersions []string
	}{
		{
			desc:         "NoChannel, ControlPlane",
			channel:      NoChannel,
			resource:     ControlPlane,
			wantDefault:  ServerConfig.DefaultClusterVersion,
			wantVersions: ServerConfig.ValidMasterVersions,
		},
		{
			desc:         "NoChannel, Node",
			channel:      NoChannel,
			resource:     Node,
			wantDefault:  ServerConfig.DefaultClusterVersion,
			wantVersions: ServerConfig.ValidNodeVersions,
		},
		{
			desc:         "Rapid, ControlPlane",
			channel:      Rapid,
			resource:     ControlPlane,
			wantDefault:  ServerConfig.Channels[0].DefaultVersion,
			wantVersions: ServerConfig.Channels[0].ValidVersions,
		},
		{
			desc:         "Rapid, Node",
			channel:      Rapid,
			resource:     Node,
			wantDefault:  ServerConfig.Channels[0].DefaultVersion,
			wantVersions: ServerConfig.Channels[0].ValidVersions,
		},
		{
			desc:         "Regular, ControlPlane",
			channel:      Regular,
			resource:     ControlPlane,
			wantDefault:  ServerConfig.Channels[1].DefaultVersion,
			wantVersions: ServerConfig.Channels[1].ValidVersions,
		},
		{
			desc:         "Regular, Node",
			channel:      Regular,
			resource:     Node,
			wantDefault:  ServerConfig.Channels[1].DefaultVersion,
			wantVersions: ServerConfig.Channels[1].ValidVersions,
		},
		{
			desc:         "Stable, ControlPlane",
			channel:      Stable,
			resource:     ControlPlane,
			wantDefault:  ServerConfig.Channels[2].DefaultVersion,
			wantVersions: ServerConfig.Channels[2].ValidVersions,
		},
		{
			desc:         "Stable, Node",
			channel:      Stable,
			resource:     Node,
			wantDefault:  ServerConfig.Channels[2].DefaultVersion,
			wantVersions: ServerConfig.Channels[2].ValidVersions,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log.StandardLogger().SetOutput(buf)

			gotDefault, gotValid := getVersions(ServerConfig, tc.channel, tc.resource)
			if diff := cmp.Diff(tc.wantDefault, gotDefault); diff != "" {
				t.Errorf("getVersions diff for default version (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantVersions, gotValid); diff != "" {
				t.Errorf("getVersions diff for valid versions (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsFormatValid(t *testing.T) {
	cases := []struct {
		desc    string
		version string
		wantErr string
	}{
		{
			desc:    "Major-Minor",
			version: "1.20",
		},
		{
			desc:    "Major-Minor-Patch",
			version: "1.20.1",
		},
		{
			desc:    "Major-Minor-Patch 2",
			version: "1.20.12",
		},
		{
			desc:    "Major-Minor-Patch-GKE",
			version: "1.20.12-gke.1",
		},
		{
			desc:    "Major-Minor-Patch-GKE 2",
			version: "1.20.1-gke.4000",
		},
		{
			desc:    "Major, no Minor",
			version: "1.",
			wantErr: "malformed",
		},
		{
			desc:    "Major only",
			version: "1",
			wantErr: "malformed",
		},
		{
			desc:    "Major-Minor, no Patch",
			version: "1.20.",
			wantErr: "malformed",
		},
		{
			desc:    "No Patch",
			version: "1.20-gke.4000",
			wantErr: "malformed",
		},
		{
			desc:    "No GKE",
			version: "1.20.1-gke.",
			wantErr: "malformed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := IsFormatValid(tc.version)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("IsFormatValid diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsVersionValid(t *testing.T) {
	cases := []struct {
		desc       string
		desired    string
		current    string
		defVersion string
		valid      []string
		wantErr    string
	}{
		{
			desc:       "Default alias, in-place",
			desired:    "-",
			current:    "1.21.1-gke.1800",
			defVersion: "1.21.1-gke.1800",
			wantErr:    "cannot upgrade in-place",
		},
		{
			desc:    "Desired not found",
			desired: "2.21.1-gke.1800",
			current: "1.21.1-gke.1800",
			valid: []string{
				"1.21.2-gke.1800",
			},
			wantErr: "was not found",
		},
		{
			desc:    "Desired is older",
			desired: "1.21.1-gke.1800",
			current: "1.21.2-gke.1800",
			valid: []string{
				"1.21.3-gke.1800",
				"1.21.2-gke.1800",
				"1.21.1-gke.1800",
			},
			wantErr: "must be newer than current version",
		},
		{
			desc:    "Desired is equal",
			desired: "1.21.2-gke.1800",
			current: "1.21.2-gke.1800",
			valid: []string{
				"1.21.3-gke.1800",
				"1.21.2-gke.1800",
				"1.21.1-gke.1800",
			},
			wantErr: "must be newer than current version",
		},
		{
			desc:    "Version patch alias, Desired is greater",
			desired: "1.21.2",
			current: "1.21.2-gke.1700",
			valid: []string{
				"1.21.2-gke.1800",
				"1.21.2-gke.1700",
				"1.21.1-gke.1800",
			},
		},
		{
			desc:    "Version minor alias, desired is greater",
			desired: "1.21",
			current: "1.21.2-gke.1700",
			valid: []string{
				"1.21.2-gke.1800",
				"1.21.2-gke.1700",
				"1.21.1-gke.1800",
			},
		},
		{
			desc:    "Version alias resolves to current",
			desired: "1.21",
			current: "1.21.2-gke.1800",
			valid: []string{
				"1.21.2-gke.1800",
				"1.21.2-gke.1700",
				"1.21.1-gke.1800",
			},
			wantErr: "must be newer than current version",
		},
		{
			desc:    "Partial gke patch matches first, desired is equal",
			desired: "1.21.1-gke.1",
			current: "1.21.1-gke.1800",
			valid: []string{
				"1.21.1-gke.1800",
				"1.21.1-gke.1700",
				"1.21.1-gke.1600",
			},
			wantErr: "must be newer than current version",
		},
		{
			desc:    "Partial gke patch matches first, desired is newer",
			desired: "1.21.1-gke.1",
			current: "1.21.1-gke.1700",
			valid: []string{
				"1.21.1-gke.1800",
				"1.21.1-gke.1700",
				"1.21.1-gke.1600",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := isVersionValid(tc.desired, tc.current, tc.defVersion, tc.valid)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("isVersionValid diff (-want +got):\n%s", diff)
			}
		})
	}
}
