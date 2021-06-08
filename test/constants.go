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

import "fmt"

const (
	ProjectName                     = "test-project"
	OperationDone                   = "DONE"
	GenericOperationName            = "operation-generic-op"
	SwitchToCustomModeOperationName = "operation-switch-mode"
	UpdateMasterOperationName       = "operation-update-master"
	UpdateNodePoolOperationName     = "operation-update-nodepool"
	ClusterName                     = "cluster-c"
	NodePoolName                    = "default-pool"
	InstanceGroupManagerName        = "default-pool-m"
	InstanceTemplateName            = "default-pool-t"
	SelectedNetwork                 = "network-0"
	RegionA                         = "region-a"
	ZoneA0                          = "region-a-0"
	ZoneA1                          = "region-a-1"
	ComputeAPI                      = "https://compute.googleapis.com/compute/v1"
	ContainerAPI                    = "https://container.googleapis.com/compute/v1"
)

var (
	InstanceGroupURL            = SelfLink(ComputeAPI, fmt.Sprintf("projects/%s/zones/%s/instanceGroups/%s", ProjectName, ZoneA0, "default-pool-g"))
	InstanceGroupManagerZoneA0  = SelfLink(ComputeAPI, fmt.Sprintf("projects/%s/zones/%s/instanceGroupManagers/%s", ProjectName, ZoneA0, InstanceGroupManagerName))
	InstanceGroupManagerZoneA1  = SelfLink(ComputeAPI, fmt.Sprintf("projects/%s/zones/%s/instanceGroupManagers/%s", ProjectName, ZoneA1, InstanceGroupManagerName))
	InstanceGroupManagerRegionA = SelfLink(ComputeAPI, fmt.Sprintf("projects/%s/regions/%s/instanceGroupManagers/%s", ProjectName, RegionA, InstanceGroupManagerName))
)
