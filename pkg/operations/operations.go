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
	"errors"
	"fmt"
	"regexp"
	"strings"
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
	Status string
	Error  string
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

func IsFinished(ctx context.Context, poll func(ctx context.Context) (OperationStatus, error)) (bool, error) {
	status, err := poll(ctx)
	if err != nil {
		return false, err
	}

	if status.Status != StatusDone {
		return false, nil
	}

	if status.Error != "" {
		return true, errors.New(status.Error)
	}
	return true, nil
}

// WaitForOperationInProgress will attempt a retry of the function.
func WaitForOperationInProgress(ctx context.Context, f func(ctx context.Context) error, wait func(ctx context.Context, op string) error) error {
	err := f(ctx)
	if err == nil {
		return nil
	}

	op := ObtainID(err)
	if op == "" {
		return err
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("Operation %s is currently", op)) {
		// Match format of errors returned by the GKE API.
		return err
	}

	log.Infof("Operation %s is in progress; wait for operation to complete: %v", op, err)

	if err := wait(ctx, op); err != nil {
		return err
	}

	log.Infof("Operation %s is complete; retrying. Retry due to: %v", op, err)

	return f(ctx)
}
