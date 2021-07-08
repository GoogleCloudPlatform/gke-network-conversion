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

func TestMigrate_Run(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case <-cancelled.Done():
	}

	cases := []struct {
		desc      string
		migrators []Migrator
		method    MethodType
		ctx       context.Context
		wantErr   string
		wantLog   string
	}{
		{
			desc: "Empty migrators",
			migrators: []Migrator{
				&FakeMigrator{},
			},
			method: MigrateMethod,
			ctx:    context.Background(),
		},
		{
			desc: "Successful run",
			migrators: []Migrator{
				&FakeMigrator{},
			},
			method: MigrateMethod,
			ctx:    context.Background(),
		},
		{
			desc: "Errored migrator",
			migrators: []Migrator{
				&FakeMigrator{
					MigrateError: errors.New("expected error"),
				},
			},
			method:  MigrateMethod,
			ctx:     context.Background(),
			wantErr: "expected error",
		},
		{
			desc: "Multiple errored migrators",
			migrators: []Migrator{
				&FakeMigrator{
					MigrateError: errors.New("expected error"),
				},
				&FakeMigrator{
					MigrateError: errors.New("expected another error"),
				},
			},
			method:  MigrateMethod,
			ctx:     context.Background(),
			wantErr: "expected another error",
		},
		{
			desc: "Single errored migrators",
			migrators: []Migrator{
				&FakeMigrator{},
				&FakeMigrator{
					MigrateError: errors.New("expected error"),
				},
			},
			method:  MigrateMethod,
			ctx:     context.Background(),
			wantErr: "expected error",
		},
		{
			desc: "Migrate, Closed context",
			migrators: []Migrator{
				&FakeMigrator{},
				&FakeMigrator{},
			},
			method:  MigrateMethod,
			ctx:     cancelled,
			wantErr: "Migrate: context canceled",
		},
		{
			desc: "Validate, Closed context",
			migrators: []Migrator{
				&FakeMigrator{},
				&FakeMigrator{},
			},
			method:  ValidateMethod,
			ctx:     cancelled,
			wantErr: "Validate: context canceled",
		},
		{
			desc: "Complete, Closed context",
			migrators: []Migrator{
				&FakeMigrator{},
				&FakeMigrator{},
			},
			method:  CompleteMethod,
			ctx:     cancelled,
			wantErr: "Complete: context canceled",
		},
		{
			desc: "Validate",
			migrators: []Migrator{
				&FakeMigrator{},
				&FakeMigrator{},
			},
			method: ValidateMethod,
			ctx:    context.Background(),
		},
		{
			desc: "Migrate",
			migrators: []Migrator{
				&FakeMigrator{},
				&FakeMigrator{},
			},
			method: MigrateMethod,
			ctx:    context.Background(),
		},
		{
			desc: "Complete",
			migrators: []Migrator{
				&FakeMigrator{},
				&FakeMigrator{},
			},
			method: CompleteMethod,
			ctx:    context.Background(),
		},
		{
			desc: "Invalid method",
			migrators: []Migrator{
				&FakeMigrator{},
				&FakeMigrator{},
			},
			method: MethodType(3),
			ctx:    context.Background(),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log.StandardLogger().SetOutput(buf)
			sem := make(chan struct{}, 1)

			got := run(tc.ctx, sem, tc.method, tc.migrators...)

			if diff := test.ErrorDiff(tc.wantErr, got); diff != "" {
				t.Fatalf("migrate.run diff (-want +got):\n%s", diff)
			}

			if diff := !strings.Contains(buf.String(), tc.wantLog); tc.wantLog != "" && diff {
				t.Errorf("migrate.run missing log output:\n\twanted entry: %s\n\tgot entries: %s", tc.wantLog, buf.String())
			}
		})
	}
}
