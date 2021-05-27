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
package pkg

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLocationParent(t *testing.T) {
	cases := []struct {
		desc     string
		project  string
		location string
		want     string
	}{
		{
			desc:     "Project-Location x-y",
			project:  "x",
			location: "y",
			want:     "projects/x/locations/y",
		},
		{
			desc:     "Project-Location z-dash",
			project:  "z",
			location: "-",
			want:     "projects/z/locations/-",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := LocationPath(tc.project, tc.location)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("output differed:\n%s", diff)
			}
		})
	}
}

func TestClusterParent(t *testing.T) {
	cases := []struct {
		desc     string
		project  string
		location string
		cluster  string
		want     string
	}{
		{
			desc:     "Project-Location-Cluster x-y-c",
			project:  "x",
			location: "y",
			cluster:  "c",
			want:     "projects/x/locations/y/clusters/c",
		},
		{
			desc:     "Project-Location-Cluster z-dash-c",
			project:  "z",
			location: "-",
			cluster:  "c",
			want:     "projects/z/locations/-/clusters/c",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := ClusterPath(tc.project, tc.location, tc.cluster)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("output differed:\n%s", diff)
			}
		})
	}
}

func TestNodePoolParent(t *testing.T) {
	cases := []struct {
		desc     string
		project  string
		location string
		cluster  string
		nodePool string
		want     string
	}{
		{
			desc:     "Project-Location-Cluster-NodePool x-y-c-p",
			project:  "x",
			location: "y",
			cluster:  "c",
			nodePool: "p",
			want:     "projects/x/locations/y/clusters/c/nodePools/p",
		},
		{
			desc:     "Project-Location-Cluster-NodePool z-dash-c-np",
			project:  "z",
			location: "-",
			cluster:  "c",
			nodePool: "np",
			want:     "projects/z/locations/-/clusters/c/nodePools/np",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := NodePoolPath(tc.project, tc.location, tc.cluster, tc.nodePool)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("output differed:\n%s", diff)
			}
		})
	}
}

func TestOperations(t *testing.T) {
	cases := []struct {
		desc      string
		project   string
		location  string
		operation string
		want      string
	}{
		{
			desc:      "Project-Location-Cluster x-y-c",
			project:   "x",
			location:  "y",
			operation: "c",
			want:      "projects/x/locations/y/operations/c",
		},
		{
			desc:      "Project-Location-Cluster z-dash-c",
			project:   "z",
			location:  "-",
			operation: "c",
			want:      "projects/z/locations/-/operations/c",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := OperationsPath(tc.project, tc.location, tc.operation)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("output differed:\n%s", diff)
			}
		})
	}
}
