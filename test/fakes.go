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
package test

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
	"legacymigration/pkg"
)

var (
	PrePatchCluster = container.Cluster{
		Name:     ClusterName,
		Location: RegionA,
		Network:  SelectedNetwork,
	}
)

type FakeMigrator struct {
	Error error
}

func (m *FakeMigrator) Migrate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("context done")
	default:
		return m.Error
	}
}

type FakeCompute struct {
	SwitchToCustomModeResp *compute.Operation
	SwitchToCustomModeErr  error

	GetGlobalOperationResp *compute.Operation
	GetGlobalOperationErr  error

	WaitOperationResp *compute.Operation
	WaitOperationErr  error
}

func (f *FakeCompute) GetInstanceGroupManager(ctx context.Context, project, zone, instanceGroupManager string, opts ...googleapi.CallOption) (*compute.InstanceGroupManager, error) {
	return nil, nil
}
func (f *FakeCompute) GetInstanceTemplate(ctx context.Context, project, instanceTemplate string, opts ...googleapi.CallOption) (*compute.InstanceTemplate, error) {
	return nil, nil
}
func (f *FakeCompute) SwitchToCustomMode(ctx context.Context, project, name string, opts ...googleapi.CallOption) (*compute.Operation, error) {
	return f.SwitchToCustomModeResp, f.SwitchToCustomModeErr
}
func (f *FakeCompute) GetGlobalOperation(ctx context.Context, project, name string, opts ...googleapi.CallOption) (*compute.Operation, error) {
	return f.GetGlobalOperationResp, f.GetGlobalOperationErr
}
func (f *FakeCompute) WaitOperation(ctx context.Context, project string, op *compute.Operation, opts ...googleapi.CallOption) (*compute.Operation, error) {
	return f.WaitOperationResp, f.WaitOperationErr
}
func (f *FakeCompute) ListNetworks(ctx context.Context, project string) (networks []*compute.Network, err error) {
	return nil, nil
}

type FakeContainer struct {
	ListClustersResp *container.ListClustersResponse
	ListClustersErr  error
}

func (f *FakeContainer) UpdateMaster(ctx context.Context, req *container.UpdateMasterRequest, opts ...googleapi.CallOption) (*container.Operation, error) {
	return nil, nil
}
func (f *FakeContainer) GetCluster(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.Cluster, error) {
	return nil, nil
}
func (f *FakeContainer) ListClusters(ctx context.Context, parent string, opts ...googleapi.CallOption) (*container.ListClustersResponse, error) {
	return f.ListClustersResp, f.ListClustersErr
}
func (f *FakeContainer) GetOperation(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.Operation, error) {
	return nil, nil
}
func (f *FakeContainer) UpdateNodePool(ctx context.Context, req *container.UpdateNodePoolRequest, opts ...googleapi.CallOption) (*container.Operation, error) {
	return nil, nil
}
func (f *FakeContainer) ListNodePools(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.ListNodePoolsResponse, error) {
	return nil, nil
}

func DefaultFakeCompute() *FakeCompute {
	switchToCustomModeOperationSelfLink := SelfLink(ContainerAPI, fmt.Sprintf("projects/%s/global/operations/%s", ProjectName, SwitchToCustomModeOperationName))
	return &FakeCompute{
		SwitchToCustomModeResp: &compute.Operation{
			Name:          SwitchToCustomModeOperationName,
			Status:        OperationDone,
			StatusMessage: "",
			SelfLink:      switchToCustomModeOperationSelfLink,
		},
		SwitchToCustomModeErr: nil,

		GetGlobalOperationResp: &compute.Operation{
			Name:          SwitchToCustomModeOperationName,
			Status:        OperationDone,
			StatusMessage: "",
			SelfLink:      switchToCustomModeOperationSelfLink,
		},
		GetGlobalOperationErr: nil,

		WaitOperationResp: &compute.Operation{
			Name:          SwitchToCustomModeOperationName,
			Status:        OperationDone,
			StatusMessage: "",
			SelfLink:      switchToCustomModeOperationSelfLink,
		},
		WaitOperationErr: nil,
	}
}

func DefaultFakeContainer() *FakeContainer {
	c := PrePatchCluster
	c.Subnetwork = "subnet"

	return &FakeContainer{
		ListClustersResp: &container.ListClustersResponse{
			Clusters: []*container.Cluster{&PrePatchCluster},
		},
		ListClustersErr: nil,
	}
}

func DefaultClients() *pkg.Clients {
	return &pkg.Clients{
		Compute:   DefaultFakeCompute(),
		Container: DefaultFakeContainer(),
	}
}
