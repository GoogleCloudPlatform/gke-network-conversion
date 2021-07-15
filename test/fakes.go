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
	"fmt"

	"legacymigration/pkg"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
)

var (
	PrePatchCluster = container.Cluster{
		Name:     ClusterName,
		Location: RegionA,
		Network:  SelectedNetwork,
		ReleaseChannel: &container.ReleaseChannel{
			Channel: Unspecified,
		},
		CurrentMasterVersion: "1.19.10-gke.1700",
	}
)

type FakeCompute struct {
	GetInstanceGroupManagerResp *compute.InstanceGroupManager
	GetInstanceGroupManagerErr  error

	GetInstanceTemplateResp *compute.InstanceTemplate
	GetInstanceTemplateErr  error

	SwitchToCustomModeResp *compute.Operation
	SwitchToCustomModeErr  error

	GetGlobalOperationResp *compute.Operation
	GetGlobalOperationErr  error

	WaitOperationResp *compute.Operation
	WaitOperationErr  error

	GetNetworkResp *compute.Network
	GetNetworkErr  error

	ListNetworksResp []*compute.Network
	ListNetworksErr  error
}

func (f *FakeCompute) GetInstanceGroupManager(ctx context.Context, project, zone, instanceGroupManager string, opts ...googleapi.CallOption) (*compute.InstanceGroupManager, error) {
	return f.GetInstanceGroupManagerResp, f.GetInstanceGroupManagerErr
}
func (f *FakeCompute) GetInstanceTemplate(ctx context.Context, project, instanceTemplate string, opts ...googleapi.CallOption) (*compute.InstanceTemplate, error) {
	return f.GetInstanceTemplateResp, f.GetInstanceTemplateErr
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
func (f *FakeCompute) GetNetwork(ctx context.Context, project, network string, opts ...googleapi.CallOption) (*compute.Network, error) {
	return f.GetNetworkResp, f.GetNetworkErr
}
func (f *FakeCompute) ListNetworks(ctx context.Context, project string) ([]*compute.Network, error) {
	return f.ListNetworksResp, f.ListNetworksErr
}

type FakeContainer struct {
	UpdateMasterResp *container.Operation
	UpdateMasterErr  error

	GetClusterResp *container.Cluster
	GetClusterErr  error

	ListClustersResp *container.ListClustersResponse
	ListClustersErr  error

	GetOperationResp *container.Operation
	GetOperationErr  error

	UpdateNodePoolResp *container.Operation
	UpdateNodePoolErr  error

	ListNodePoolsResp *container.ListNodePoolsResponse
	ListNodePoolsErr  error

	GetServerConfigResp *container.ServerConfig
	GetServerConfigErr  error
}

func (f *FakeContainer) UpdateMaster(ctx context.Context, req *container.UpdateMasterRequest, opts ...googleapi.CallOption) (*container.Operation, error) {
	return f.UpdateMasterResp, f.UpdateMasterErr
}
func (f *FakeContainer) GetCluster(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.Cluster, error) {
	return f.GetClusterResp, f.GetClusterErr
}
func (f *FakeContainer) ListClusters(ctx context.Context, parent string, opts ...googleapi.CallOption) (*container.ListClustersResponse, error) {
	return f.ListClustersResp, f.ListClustersErr
}
func (f *FakeContainer) GetOperation(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.Operation, error) {
	return f.GetOperationResp, f.GetOperationErr
}
func (f *FakeContainer) UpdateNodePool(ctx context.Context, req *container.UpdateNodePoolRequest, opts ...googleapi.CallOption) (*container.Operation, error) {
	return f.UpdateNodePoolResp, f.UpdateNodePoolErr
}
func (f *FakeContainer) ListNodePools(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.ListNodePoolsResponse, error) {
	return f.ListNodePoolsResp, f.ListNodePoolsErr
}
func (f *FakeContainer) GetServerConfig(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.ServerConfig, error) {
	return f.GetServerConfigResp, f.GetServerConfigErr
}

func DefaultFakeCompute() *FakeCompute {
	switchToCustomModeOperationSelfLink := SelfLink(ContainerAPI, fmt.Sprintf("projects/%s/global/operations/%s", ProjectName, SwitchToCustomModeOperationName))
	return &FakeCompute{
		GetInstanceGroupManagerResp: &compute.InstanceGroupManager{
			Name: InstanceGroupManagerName,
		},
		GetInstanceGroupManagerErr: nil,

		GetInstanceTemplateResp: &compute.InstanceTemplate{
			Name: InstanceTemplateName,
			Properties: &compute.InstanceProperties{
				NetworkInterfaces: []*compute.NetworkInterface{
					{},
				},
			},
		},
		GetInstanceTemplateErr: nil,

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

		GetNetworkResp: &compute.Network{
			Name: SelectedNetwork,
		},
		GetNetworkErr: nil,

		ListNetworksResp: []*compute.Network{
			{
				Name: SelectedNetwork,
			},
		},
		ListNetworksErr: nil,
	}
}

func DefaultFakeContainer() *FakeContainer {
	c := PrePatchCluster
	c.Subnetwork = "subnet"

	return &FakeContainer{
		UpdateMasterResp: &container.Operation{
			Name:          UpdateMasterOperationName,
			Location:      RegionA,
			Status:        OperationDone,
			StatusMessage: "",
			SelfLink:      SelfLink(ContainerAPI, pkg.OperationsPath(ProjectName, RegionA, UpdateMasterOperationName)),
		},
		UpdateMasterErr: nil,

		GetClusterResp: &c,
		GetClusterErr:  nil,

		ListClustersResp: &container.ListClustersResponse{
			Clusters: []*container.Cluster{&PrePatchCluster},
		},
		ListClustersErr: nil,

		GetOperationResp: &container.Operation{
			Name:          GenericOperationName,
			Location:      RegionA,
			Status:        OperationDone,
			StatusMessage: "",
			SelfLink:      SelfLink(ContainerAPI, pkg.OperationsPath(ProjectName, RegionA, GenericOperationName)),
		},
		GetOperationErr: nil,

		UpdateNodePoolResp: &container.Operation{
			Name:          UpdateNodePoolOperationName,
			Location:      RegionA,
			Status:        OperationDone,
			StatusMessage: "",
			SelfLink:      SelfLink(ContainerAPI, pkg.OperationsPath(ProjectName, RegionA, UpdateNodePoolOperationName)),
		},
		UpdateNodePoolErr: nil,

		ListNodePoolsResp: &container.ListNodePoolsResponse{
			NodePools: []*container.NodePool{
				{
					Name: NodePoolName,
					InstanceGroupUrls: []string{
						InstanceGroupURL,
					},
				},
			},
		},
		ListNodePoolsErr: nil,

		GetServerConfigResp: &container.ServerConfig{
			Channels: []*container.ReleaseChannelConfig{
				{
					Channel:        Rapid,
					DefaultVersion: "1.20.6-gke.1400",
					ValidVersions: []string{
						"1.21.1-gke.1800",
						"1.20.7-gke.1800",
						"1.20.6-gke.1400",
					},
				}, {
					Channel:        Regular,
					DefaultVersion: "1.19.10-gke.1600",
					ValidVersions: []string{
						"1.20.6-gke.1000",
						"1.19.10-gke.1700",
						"1.19.10-gke.1600",
					},
				}, {
					Channel:        Stable,
					DefaultVersion: "1.18.17-gke.1901",
					ValidVersions: []string{
						"1.19.10-gke.1000",
						"1.18.18-gke.1100",
						"1.18.17-gke.1901",
					},
				},
			},
			DefaultClusterVersion: "1.19.10-gke.1600",
			ValidMasterVersions: []string{
				"1.20.7-gke.1800",
				"1.20.6-gke.1000",
				"1.19.11-gke.1700",
				"1.19.10-gke.1700",
			},
			ValidNodeVersions: []string{
				"1.20.7-gke.1800",
				"1.20.6-gke.1000",
				"1.19.11-gke.1700",
				"1.19.10-gke.1700",
			},
		},
		GetServerConfigErr: nil,
	}
}

func DefaultClients() *pkg.Clients {
	return &pkg.Clients{
		Compute:   DefaultFakeCompute(),
		Container: DefaultFakeContainer(),
	}
}
