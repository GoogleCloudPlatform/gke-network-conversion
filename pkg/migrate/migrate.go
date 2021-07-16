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
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"go.uber.org/multierr"
)

type Migrator interface {
	Complete(ctx context.Context) error
	Validate(ctx context.Context) error
	Migrate(ctx context.Context) error
	ResourcePath() string
}

type MethodType int

const (
	CompleteMethod MethodType = iota
	ValidateMethod
	MigrateMethod
)

func (e MethodType) String() string {
	switch e {
	case CompleteMethod:
		return "Complete"
	case ValidateMethod:
		return "Validate"
	case MigrateMethod:
		return "Migrate"
	default:
		return fmt.Sprintf("%d", int(e))
	}
}

// Complete runs Migrator.Complete on all migrators.
func Complete(ctx context.Context, sem chan struct{}, migrators ...Migrator) error {
	return run(ctx, sem, CompleteMethod, migrators...)
}

// Validate runs Migrator.Validate on all migrators.
func Validate(ctx context.Context, sem chan struct{}, migrators ...Migrator) error {
	return run(ctx, sem, ValidateMethod, migrators...)
}

// Migrate runs Migrator.Migrate on all migrators.
func Migrate(ctx context.Context, sem chan struct{}, migrators ...Migrator) error {
	return run(ctx, sem, MigrateMethod, migrators...)
}

// run rate-limits the execution of a specified Migrator method based on the incoming semaphore.
// Accumulates any errors into a single error.
func run(ctx context.Context, sem chan struct{}, t MethodType, migrators ...Migrator) error {
	var (
		errors  error
		wg      = sync.WaitGroup{}
		results = make(chan error, len(migrators))
	)

Loop:
	for _, m := range migrators {
		select {
		case <-ctx.Done():
			errors = multierr.Append(errors, fmt.Errorf("context closed during %T.%v: %w", m, t, ctx.Err()))
			break Loop
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(m Migrator) {
			defer func() { <-sem }()
			defer wg.Done()
			var method func(ctx context.Context) error
			switch t {
			case CompleteMethod:
				method = m.Complete
			case ValidateMethod:
				method = m.Validate
			case MigrateMethod:
				method = m.Migrate
			default:
				log.Errorf("Invalid method %v", t)
				return
			}
			results <- method(ctx)
		}(m)
	}
	wg.Wait()
	close(results)

	if errors != nil {
		return errors
	}

	for err := range results {
		if err != nil {
			errors = multierr.Append(errors, err)
		}
	}

	return errors
}
