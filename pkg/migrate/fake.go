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
	"context"
	"errors"
)

type FakeMigrator struct {
	CompleteError error
	ValidateError error
	MigrateError  error
}

func (m *FakeMigrator) Complete(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("context canceled")
	default:
		return m.CompleteError
	}
}

func (m *FakeMigrator) Validate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("context canceled")
	default:
		return m.ValidateError
	}
}

func (m *FakeMigrator) Migrate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("context canceled")
	default:
		return m.MigrateError
	}
}

func (m *FakeMigrator) ResourcePath() string {
	return "resource-path"
}
