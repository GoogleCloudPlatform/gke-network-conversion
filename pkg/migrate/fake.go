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
