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

var (
	ServerConfig = &container.ServerConfig{
		Channels: []*container.ReleaseChannelConfig{
			{
				Channel:        test.Rapid,
				DefaultVersion: "1.20.6-gke.1400",
				ValidVersions: []string{
					"1.21.1-gke.1800",
					"1.20.7-gke.1800",
					"1.20.6-gke.1400",
				},
			}, {
				Channel:        test.Regular,
				DefaultVersion: "1.19.10-gke.1600",
				ValidVersions: []string{
					"1.20.6-gke.1000",
					"1.19.10-gke.1700",
					"1.19.10-gke.1600",
				},
			}, {
				Channel:        test.Stable,
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
			desc:         "Unspecified, ControlPlane",
			channel:      test.Unspecified,
			resource:     ControlPlane,
			wantDefault:  ServerConfig.DefaultClusterVersion,
			wantVersions: ServerConfig.ValidMasterVersions,
		},
		{
			desc:         "Unspecified, Node",
			channel:      test.Unspecified,
			resource:     Node,
			wantDefault:  ServerConfig.DefaultClusterVersion,
			wantVersions: ServerConfig.ValidNodeVersions,
		},
		{
			desc:         "Rapid, ControlPlane",
			channel:      test.Rapid,
			resource:     ControlPlane,
			wantDefault:  ServerConfig.Channels[0].DefaultVersion,
			wantVersions: ServerConfig.Channels[0].ValidVersions,
		},
		{
			desc:         "Rapid, Node",
			channel:      test.Rapid,
			resource:     Node,
			wantDefault:  ServerConfig.Channels[0].DefaultVersion,
			wantVersions: ServerConfig.Channels[0].ValidVersions,
		},
		{
			desc:         "Regular, ControlPlane",
			channel:      test.Regular,
			resource:     ControlPlane,
			wantDefault:  ServerConfig.Channels[1].DefaultVersion,
			wantVersions: ServerConfig.Channels[1].ValidVersions,
		},
		{
			desc:         "Regular, Node",
			channel:      test.Regular,
			resource:     Node,
			wantDefault:  ServerConfig.Channels[1].DefaultVersion,
			wantVersions: ServerConfig.Channels[1].ValidVersions,
		},
		{
			desc:         "Stable, ControlPlane",
			channel:      test.Stable,
			resource:     ControlPlane,
			wantDefault:  ServerConfig.Channels[2].DefaultVersion,
			wantVersions: ServerConfig.Channels[2].ValidVersions,
		},
		{
			desc:         "Stable, Node",
			channel:      test.Stable,
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
			desc:    "Empty",
			wantErr: "malformed version: version must not be empty",
		},
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
			desc:    "Major version 2",
			version: "2.20.1-gke.4000",
			wantErr: "not compatible with major versions other than 1",
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

func TestIsUpgrade(t *testing.T) {
	cases := []struct {
		desc         string
		desired      string
		current      string
		valid        []string
		allowInPlace bool
		wantErr      string
	}{
		{
			desc:    "Desired not found",
			desired: "1.21.2-gke.1800",
			current: "1.21.1-gke.1800",
			valid: []string{
				"1.21.1-gke.1800",
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
			desc:    "Desired is equal, do not allow in-place",
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
			desc:         "Desired is equal, allow in-place",
			desired:      "1.21.2-gke.1800",
			current:      "1.21.2-gke.1800",
			allowInPlace: true,
			valid: []string{
				"1.21.3-gke.1800",
				"1.21.2-gke.1800",
				"1.21.1-gke.1800",
			},
		},
		{
			desc:    "Current version no longer valid",
			desired: "1.21.1-gke.1800",
			current: "1.21.1-gke.1500",
			valid: []string{
				"1.21.1-gke.1800",
				"1.21.1-gke.1700",
				"1.21.1-gke.1600",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := isUpgrade(tc.desired, tc.current, tc.valid, tc.allowInPlace)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("isUpgrade diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsWithinVersionSkew(t *testing.T) {
	cases := []struct {
		desc      string
		npVersion string
		cpVersion string
		skew      int
		wantErr   string
	}{
		{
			desc:      "Same version",
			npVersion: "1.21.2-gke.1800",
			cpVersion: "1.21.1-gke.1800",
			skew:      MaxVersionSkew,
		},
		{
			desc:      "Node pool within version skew",
			npVersion: "1.22",
			cpVersion: "1.21",
			skew:      MaxVersionSkew,
		},
		{
			desc:      "Node pool beyond version skew",
			npVersion: "1.23",
			cpVersion: "1.21",
			skew:      MaxVersionSkew,
			wantErr:   "must be within 1 minor versions of desired control plane version",
		},
		{
			desc:      "Node pool within version skew",
			npVersion: "1.21.2-gke.1800",
			cpVersion: "1.21.1-gke.1800",
			skew:      2,
		},
		{
			desc:      "Node pool within version skew",
			npVersion: "1.21",
			cpVersion: "1.22",
			skew:      2,
		},
		{
			desc:      "Node pool within version skew",
			npVersion: "1.21.2-gke.1800",
			cpVersion: "1.23",
			skew:      2,
		},
		{
			desc:      "Node pool beyond version skew",
			npVersion: "1.21",
			cpVersion: "1.24",
			skew:      2,
			wantErr:   "must be within 2 minor versions of desired control plane version",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := IsWithinVersionSkew(tc.npVersion, tc.cpVersion, tc.skew)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("IsWithinVersionSkew diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveVersion(t *testing.T) {
	cases := []struct {
		desc    string
		desired string
		defalt  string
		valid   []string
		want    string
		wantErr string
	}{
		{
			desc:    "Version list is empty",
			desired: "-",
			defalt:  "1.21.1-gke.1800",
			valid:   []string{},
			wantErr: "list of valid versions is empty",
		},
		{
			desc:    "Default is missing",
			desired: "-",
			valid: []string{
				"1.21.2-gke.1800",
			},
			wantErr: "default version is missing",
		},
		{
			desc:    "Default alias",
			desired: "-",
			defalt:  "1.21.1-gke.1800",
			valid: []string{
				"1.21.2-gke.1800",
				"1.21.1-gke.1800",
			},
			want: "1.21.1-gke.1800",
		},
		{
			desc:    "Default latest",
			desired: "latest",
			defalt:  "unused",
			valid: []string{
				"1.21.2-gke.1800",
				"1.21.1-gke.1800",
			},
			want: "1.21.2-gke.1800",
		},
		{
			desc:    "Full version",
			desired: "1.21.1-gke.1800",
			defalt:  "unused",
			valid: []string{
				"1.21.2-gke.1800",
				"1.21.1-gke.1800",
			},
			want: "1.21.1-gke.1800",
		},
		{
			desc:    "Full version not found",
			desired: "1.20.2-gke.1800",
			defalt:  "unused",
			valid: []string{
				"1.21.2-gke.1800",
				"1.21.1-gke.1800",
			},
			wantErr: "could not be resolved",
		},
		{
			desc:    "GKE version resolved",
			desired: "1.21.1-gke.1",
			defalt:  "unused",
			valid: []string{
				"1.21.1-gke.1900",
				"1.21.1-gke.1800",
			},
			want: "1.21.1-gke.1900",
		},
		{
			desc:    "Patch version resolved",
			desired: "1.21.2",
			defalt:  "unused",
			valid: []string{
				"1.21.2-gke.1800",
				"1.21.1-gke.1800",
			},
			want: "1.21.2-gke.1800",
		},
		{
			desc:    "Minor version resolved",
			desired: "1.21",
			defalt:  "unused",
			valid: []string{
				"1.22.1-gke.1800",
				"1.21.1-gke.1800",
			},
			want: "1.21.1-gke.1800",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := resolveVersion(tc.desired, tc.defalt, tc.valid)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("isUpgrade diff (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("getVersions diff for default version (-want +got):\n%s", diff)
			}
		})
	}
}
