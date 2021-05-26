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
package pkg

import (
	"context"
	"fmt"
	"regexp"

	computealpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
)

const (
	LocationParentPath = "projects/%s/locations/%s"
	ClusterParentPath  = LocationParentPath + "/clusters/%s"
	RelativeNodePoolPath = "/nodePools/%s"
	NodePoolParentPath = ClusterParentPath + RelativeNodePoolPath
	OperationParent    = LocationParentPath + "/operations/%s"

	DefaultVersion								 = "-"
	AnyLocation										 = "-"
)

var (
	zoneFormat = regexp.MustCompile(`\w+-\w+-\w`)
)

type ComputeService interface {
	GetInstanceGroupManager(ctx context.Context, project, location, instanceGroupManager string, opts ...googleapi.CallOption) (*compute.InstanceGroupManager, error)
	GetInstanceTemplate(ctx context.Context, project, name string, opts ...googleapi.CallOption) (*compute.InstanceTemplate, error)
	GetGlobalOperation(ctx context.Context, project, name string, opts ...googleapi.CallOption) (*compute.Operation, error)
	WaitOperation(ctx context.Context, project string, op *compute.Operation, opts ...googleapi.CallOption) (*compute.Operation, error)
	SwitchToCustomMode(ctx context.Context, project, name string, opts ...googleapi.CallOption) (*compute.Operation, error)
	ListNetworks(ctx context.Context, project string) ([]*compute.Network, error)
}

type ContainerService interface {
	UpdateMaster(ctx context.Context, req *container.UpdateMasterRequest, opts ...googleapi.CallOption) (*container.Operation, error)
	GetCluster(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.Cluster, error)
	ListClusters(ctx context.Context, parent string, opts ...googleapi.CallOption) (*container.ListClustersResponse, error)

	GetOperation(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.Operation, error)

	UpdateNodePool(ctx context.Context, req *container.UpdateNodePoolRequest, opts ...googleapi.CallOption) (*container.Operation, error)
	ListNodePools(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.ListNodePoolsResponse, error)
}

type Compute struct{
	V1    *compute.Service
	Alpha *computealpha.Service
}

func (c *Compute) GetInstanceGroupManager(ctx context.Context, project, location, instanceGroupManager string, opts ...googleapi.CallOption) (*compute.InstanceGroupManager, error) {
	if IsZonal(location) {
		return c.V1.InstanceGroupManagers.Get(project, location, instanceGroupManager).Context(ctx).Do(opts...)
	}
	return c.V1.RegionInstanceGroupManagers.Get(project, location, instanceGroupManager).Context(ctx).Do(opts...)
}

func (c *Compute) GetInstanceTemplate (ctx context.Context, project, instanceTemplate string, opts ...googleapi.CallOption) (*compute.InstanceTemplate, error) {
	return c.V1.InstanceTemplates.Get(project, instanceTemplate).Context(ctx).Do(opts...)
}

// SwitchToCustomMode transparently uses computealpha.Service.SwitchToCustomMode.
func (c *Compute) SwitchToCustomMode(ctx context.Context, project, name string, opts ...googleapi.CallOption) (*compute.Operation, error){
	resp, err := c.Alpha.Networks.SwitchToCustomMode(project, name).Context(ctx).Do(opts...)
	if err != nil {
		return nil, err
	}
	return c.GetGlobalOperation(ctx, project, resp.Name, opts...)
}

func (c *Compute) GetGlobalOperation(ctx context.Context, project, name string, opts ...googleapi.CallOption) (*compute.Operation, error){
	return c.V1.GlobalOperations.Get(project, name).Context(ctx).Do(opts...)
}

func (c *Compute) WaitOperation(ctx context.Context, project string, op *compute.Operation, opts ...googleapi.CallOption) (*compute.Operation, error){
	switch {
	case op.Zone != "":
		return c.V1.ZoneOperations.Wait(project, op.Zone, op.Name).Context(ctx).Do(opts...)
	case op.Region != "":
		return c.V1.RegionOperations.Wait(project, op.Region, op.Name).Context(ctx).Do(opts...)
	default:
		return c.V1.GlobalOperations.Wait(project, op.Name).Context(ctx).Do(opts...)
	}
}

func (c *Compute) ListNetworks(ctx context.Context, project string) (networks []*compute.Network, err error) {
	networks = make([]*compute.Network, 0)
	req := c.V1.Networks.List(project)
	err = req.Pages(ctx, func(page *compute.NetworkList) error {
		for _, network := range page.Items {
			networks = append(networks, network)
		}
		return nil
	})
	return
}

type Container struct{
	V1    *container.Service
}

func (c *Container) UpdateMaster(ctx context.Context, req *container.UpdateMasterRequest, opts ...googleapi.CallOption) (*container.Operation, error){
	return c.V1.Projects.Locations.Clusters.UpdateMaster(req.Name, req).Context(ctx).Do(opts...)
}
func (c *Container) GetCluster(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.Cluster, error){
	return c.V1.Projects.Locations.Clusters.Get(name).Context(ctx).Do(opts...)
}
func (c *Container) ListClusters(ctx context.Context, parent string, opts ...googleapi.CallOption) (*container.ListClustersResponse, error){
	return c.V1.Projects.Locations.Clusters.List(parent).Context(ctx).Do(opts...)
}
func (c *Container) GetOperation(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.Operation, error){
	return c.V1.Projects.Locations.Operations.Get(name).Context(ctx).Do(opts...)
}
func (c *Container) UpdateNodePool(ctx context.Context, req *container.UpdateNodePoolRequest, opts ...googleapi.CallOption) (*container.Operation, error){
	return c.V1.Projects.Locations.Clusters.NodePools.Update(req.Name, req).Context(ctx).Do(opts...)
}
func (c *Container) ListNodePools(ctx context.Context, name string, opts ...googleapi.CallOption) (*container.ListNodePoolsResponse, error){
	return c.V1.Projects.Locations.Clusters.NodePools.List(name).Context(ctx).Do(opts...)
}

type Clients struct {
	Compute      ComputeService
	Container		 ContainerService
}

func LocationParent(project, location string) string {
	return fmt.Sprintf(LocationParentPath, project, location)
}

func ClusterParent(project, location, cluster string) string {
	return fmt.Sprintf(ClusterParentPath,  project, location, cluster)
}

func NodePoolParent(project, location, cluster, name string) string {
	return fmt.Sprintf(NodePoolParentPath, project, location, cluster, name)
}

func OperationsParent(project, location, name string) string {
	return fmt.Sprintf(OperationParent, project, location, name)
}

func IsZonal(location string) bool {
	return zoneFormat.MatchString(location)
}