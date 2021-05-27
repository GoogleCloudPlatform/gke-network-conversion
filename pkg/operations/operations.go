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
package operations

import (
	"context"
	"fmt"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	StatusDone = "DONE"
)

var (
	operationRegex = regexp.MustCompile(`operation-[\d\w]+-[\d\w]+`)
)

// Operation is an interface for GCE and GKE Operations.
type Operation interface {
	// IsFinished checks whether the Operation status is complete.
	IsFinished(ctx context.Context) (bool, error)

	// String returns the Operation ID (i.e. the path).
	// Note: this Operation ID is relative to the Service.
	String() string
}

// OperationStatus is a distillation of a GCP Operation status (which vary by API).
type OperationStatus struct {
	Name   string
	Status string
	Error  string
}

// Wrapper contains distilled status information for a GCP Operation.
type Wrapper struct {
	ProjectID string
	Path      string

	// Operation holds the struct.
	// This may be referenced in the provided Poll function.
	Operation interface{}

	// Poll fetches the operation and converts the Operation's status to an OperationStatus.
	// A function allows for having only a single Operation implementation,
	// rather than per-operation type.
	Poll func(ctx context.Context, o *Wrapper) (OperationStatus, error)
}

// IsFinished determines whether the underlying operation has completed.
func (o *Wrapper) IsFinished(ctx context.Context) (bool, error) {
	status, err := o.Poll(ctx, o)
	if err != nil {
		return false, fmt.Errorf("error polling for %s: %w", o.String(), err)
	}

	if status.Status != StatusDone {
		return false, nil
	}

	if status.Error != "" {
		return false, fmt.Errorf("error(s) for Operation %s: %s", o.String(), status.Error)
	}
	return true, nil
}

func (o *Wrapper) String() string {
	return o.Path
}

type Handler interface {
	Wait(ctx context.Context, op Operation) error
}

// HandlerImpl is a thread-safe implementation of Handler.
type HandlerImpl struct {
	interval time.Duration
	deadline time.Duration
}

func NewHandler(interval time.Duration, deadline time.Duration) *HandlerImpl {
	return &HandlerImpl{interval: interval, deadline: deadline}
}

// Wait loops over Operation.IsFinished method until the operation is complete.
func (h HandlerImpl) Wait(ctx context.Context, op Operation) error {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(h.deadline))
	defer cancel()
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context error: %w", ctx.Err())
		case <-ticker.C:
			log.Debugf("Polling for %s", op)
			done, err := op.IsFinished(ctx)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

// ObtainID attempts to retrieve an operation name from the error.
func ObtainID(err error) string {
	return operationRegex.FindString(err.Error())
}
