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
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
	"legacymigration/test"
)

func TestNodePoolMigrator_Migrate(t *testing.T) {
	cases := []struct {
		desc    string
		ctx     context.Context
		m       *nodePoolMigrator
		wantErr string
		wantLog string
	}{
		// further tests incoming in another pull.
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

func TestParseSubmatch(t *testing.T) {
	cases := []struct {
		desc      string
		re        *regexp.Regexp
		substring string
		want      []string
		ok        bool
	}{
		{
			desc:      "InstanceGroupManager ZoneA0",
			re:        instanceGroupManagerRegex,
			substring: test.InstanceGroupManagerZoneA0,
			want:      []string{test.ZoneA0, test.InstanceGroupManagerName},
			ok:        true,
		},
		{
			desc:      "InstanceGroupManager ZoneA1",
			re:        instanceGroupManagerRegex,
			substring: test.InstanceGroupManagerZoneA1,
			want:      []string{test.ZoneA1, test.InstanceGroupManagerName},
			ok:        true,
		},
		{
			desc:      "InstanceGroupManager RegionA",
			re:        instanceGroupManagerRegex,
			substring: test.InstanceGroupManagerRegionA,
			want:      []string{test.RegionA, test.InstanceGroupManagerName},
			ok:        true,
		},
		{
			desc:      "InstanceGroupManager x",
			re:        instanceGroupManagerRegex,
			substring: fmt.Sprintf("projects/%s/zones/%s/instanceGroupManagers/%s", test.ProjectName, test.ZoneA0, "x"),
			want:      []string{test.ZoneA0, "x"},
			ok:        true,
		},
		{
			desc:      "Cluster path - no match",
			re:        instanceGroupManagerRegex,
			substring: fmt.Sprintf("projects/%s/zones/%s/clusters/%s", test.ProjectName, test.ZoneA0, "x"),
			ok:        false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, ok := parseSubmatch(tc.re, tc.substring)
			if ok != tc.ok {
				t.Fatalf("parseSubmatch returned not ok for input: %v, %s", tc.re, tc.substring)
			}
			if !ok {
				return
			}
			if diff := cmp.Diff(tc.want, got[1:]); diff != "" {
				t.Errorf("parseSubmatch diff (-want +got):\n%s", diff)
			}
		})
	}
}
