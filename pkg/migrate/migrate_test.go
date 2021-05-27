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
package migrate

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"legacymigration/test"
)

func TestMigrate(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case <-cancelled.Done():
	}

	cases := []struct {
		desc      string
		migrators []Migrator
		ctx       context.Context
		wantErr   string
		wantLog   string
	}{
		{
			desc: "Empty migrators",
			migrators: []Migrator{
				&test.FakeMigrator{},
			},
			ctx:     context.Background(),
			wantErr: "",
		},
		{
			desc: "Successful run",
			migrators: []Migrator{
				&test.FakeMigrator{},
			},
			ctx:     context.Background(),
			wantErr: "",
		},
		{
			desc: "Errored migrator",
			migrators: []Migrator{
				&test.FakeMigrator{
					Error: errors.New("expected error"),
				},
			},
			ctx:     context.Background(),
			wantLog: "expected error",
		},
		{
			desc: "Closed context",
			migrators: []Migrator{
				&test.FakeMigrator{},
				&test.FakeMigrator{},
			},
			ctx:     cancelled,
			wantErr: "context error: context canceled",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log.StandardLogger().SetOutput(buf)
			sem := make(chan struct{}, 1)

			got := Run(tc.ctx, sem, tc.migrators...)

			if diff := test.ErrorDiff(tc.wantErr, got); diff != "" {
				t.Fatalf("migrate.Run diff (-want +got):\n%s", diff)
			}

			if diff := !strings.Contains(buf.String(), tc.wantLog); tc.wantLog != "" && diff {
				t.Errorf("migrate.Run missing log output:\n\twanted entry: %s\n\tgot entries: %s", tc.wantLog, buf.String())
			}
		})
	}
}
