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
)

type Migrator interface {
	Migrate(ctx context.Context) error
}

// Run rate-limits the execution of Migrators based on the incoming semaphore.
// An error which occurs during the Migrator.Migrate call will cease progressing
// on that branch of the execution tree.
func Run(ctx context.Context, sem chan struct{}, migrators ...Migrator) error {
	var (
		err error
		wg  = sync.WaitGroup{}
	)

Loop:
	for _, m := range migrators {
		select {
		case <-ctx.Done():
			log.Errorf("Context closed. Waiting for pending migrators to finish.")
			err = fmt.Errorf("context error: %w", ctx.Err())
			break Loop
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(m Migrator) {
			if err := m.Migrate(ctx); err != nil {
				log.Errorf("Error during migration for %T: %v", m, err)
			}
			<-sem
			wg.Done()
		}(m)
	}
	wg.Wait()
	return err
}
