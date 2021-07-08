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
	"testing"
	"time"

	"legacymigration/test"
)

type FakeOperation struct {
	// At least one response struct should be {true, nil} or {_, !nil}.
	Responses []struct {
		finished bool
		err      error
	}

	// internal
	done  bool
	err   error
	index int
}

func (f *FakeOperation) IsFinished(_ context.Context) (bool, error) {
	// Return cached result if Operation is done.
	if f.done {
		return f.done, f.err
	}
	if f.index > len(f.Responses)-1 {
		return false, errors.New("test error")
	}
	f.done = f.Responses[f.index].finished
	f.err = f.Responses[f.index].err
	f.index++
	return f.done, f.err
}

func (f *FakeOperation) String() string {
	return "fake-op"
}

func TestWait(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()

	cases := []struct {
		desc    string
		ctx     context.Context
		op      Operation
		h       Handler
		wantErr string
	}{
		{
			desc: "First poll",
			ctx:  ctx,
			op: &FakeOperation{
				Responses: []struct {
					finished bool
					err      error
				}{
					{finished: true},
				},
			},
			h: HandlerImpl{
				interval: 1 * time.Microsecond,
				deadline: 1 * time.Second,
			},
			wantErr: "",
		},
		{
			desc: "Second poll",
			ctx:  ctx,
			op: &FakeOperation{
				Responses: []struct {
					finished bool
					err      error
				}{
					{},
					{finished: true},
				},
			},
			h: HandlerImpl{
				interval: 1 * time.Microsecond,
				deadline: 1 * time.Second,
			},
			wantErr: "",
		},
		{
			desc: "Error",
			ctx:  ctx,
			op: &FakeOperation{
				Responses: []struct {
					finished bool
					err      error
				}{
					{},
					{err: errors.New("operation get error")},
				},
			},
			h: HandlerImpl{
				interval: 1 * time.Microsecond,
				deadline: 1 * time.Second,
			},
			wantErr: "operation get error",
		},
		{
			desc: "deadline exceeded",
			ctx:  ctx,
			op: &FakeOperation{
				Responses: []struct {
					finished bool
					err      error
				}{
					{finished: true},
				},
			},
			h: HandlerImpl{
				interval: 2 * time.Microsecond,
				deadline: 1 * time.Microsecond,
			},
			wantErr: "deadline",
		},
		{
			desc: "Context cancelled",
			ctx:  cancelled,
			op: &FakeOperation{
				Responses: []struct {
					finished bool
					err      error
				}{
					{finished: true},
				},
			},
			h: HandlerImpl{
				interval: 2 * time.Microsecond,
				deadline: 1 * time.Millisecond,
			},
			wantErr: "context error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.h.Wait(tc.ctx, tc.op)
			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("HandlerImpl.Wait diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsFinished(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cases := []struct {
		desc    string
		ctx     context.Context
		poll    func(ctx context.Context) (OperationStatus, error)
		want    bool
		wantErr string
	}{
		{
			desc: "operation finished",
			ctx:  ctx,
			poll: func(ctx context.Context) (OperationStatus, error) {
				return OperationStatus{Status: StatusDone}, nil
			},
			want: true,
		},
		{
			desc: "operation not finished",
			ctx:  ctx,
			poll: func(ctx context.Context) (OperationStatus, error) {
				return OperationStatus{Status: "Not Finished"}, nil
			},
			want: false,
		},
		{
			desc: "operation error",
			ctx:  ctx,
			poll: func(ctx context.Context) (OperationStatus, error) {
				return OperationStatus{Status: StatusDone, Error: "operation error"}, nil
			},
			want:    true,
			wantErr: "operation error",
		},
		{
			desc: "poll error",
			ctx:  ctx,
			poll: func(ctx context.Context) (OperationStatus, error) {
				return OperationStatus{}, errors.New("polling error")
			},
			want:    false,
			wantErr: "polling error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := IsFinished(tc.ctx, tc.poll)

			if got != tc.want {
				t.Errorf("IsFinished diff; wanted: %v, got: %v", tc.want, got)
			}

			if diff := test.ErrorDiff(tc.wantErr, err); diff != "" {
				t.Errorf("HandlerImpl.Wait diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObtainID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		desc string
		err  error
		want string
	}{
		{
			desc: "Empty error",
			err:  errors.New(""),
			want: "",
		},
		{
			desc: "Match digits",
			err:  errors.New("operation-1234-1234"),
			want: "operation-1234-1234",
		},
		{
			desc: "Match chars",
			err:  errors.New("operation-asdf-asdf"),
			want: "operation-asdf-asdf",
		},
		{
			desc: "Match alphanumeric",
			err:  errors.New("operation-a1s2d3f4-5w6a7s8d"),
			want: "operation-a1s2d3f4-5w6a7s8d",
		},
		{
			desc: "In progress response",
			err:  errors.New("Operation operation-1622566230187-eed3f066 is currently upgrading cluster name-here. Please wait and try again once it is done."),
			want: "operation-1622566230187-eed3f066",
		},
		{
			desc: "Malformed operation",
			err:  errors.New("Operation operation-1622566230187- is currently upgrading cluster name-here. Please wait and try again once it is done."),
			want: "",
		},
		{
			desc: "No operation",
			err:  errors.New("some other error"),
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := ObtainID(tc.err)
			if got != tc.want {
				t.Errorf("ObtainID diff; wanted: %q, got: %q", tc.want, got)
			}
		})
	}
}
