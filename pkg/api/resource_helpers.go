/*
Copyright 2014 The Kubernetes Authors.

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

package api

import (
	"strings"

	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

// Returns string version of ResourceName.
func (self ResourceName) String() string {
	return string(self)
}

// Returns the CPU limit if specified.
func (self *ResourceList) Cpu() *resource.Quantity {
	if val, ok := (*self)[ResourceCPU]; ok {
		return &val
	}
	return &resource.Quantity{Format: resource.DecimalSI}
}

// Returns the Memory limit if specified.
func (self *ResourceList) Memory() *resource.Quantity {
	if val, ok := (*self)[ResourceMemory]; ok {
		return &val
	}
	return &resource.Quantity{Format: resource.BinarySI}
}

func (self *ResourceList) Pods() *resource.Quantity {
	if val, ok := (*self)[ResourcePods]; ok {
		return &val
	}
	return &resource.Quantity{}
}

func (self *ResourceList) NvidiaGPU() *resource.Quantity {
	if val, ok := (*self)[ResourceNvidiaGPU]; ok {
		return &val
	}
	return &resource.Quantity{}
}

func (self *ResourceList) NvidiaGPUMemory() *resource.Quantity {
	if val, ok := (*self)[ResourceNvidiaGPUMemory]; ok {
		return &val
	}
	return &resource.Quantity{Format: resource.BinarySI}
}

func (self *ResourceList) MemoryOfEachNvidiaGPU() []*resource.Quantity {
	ret := make([]*resource.Quantity, 0)
	for k, v := range *self {
		if strings.HasPrefix(k.String(), "/dev/nvidia") {
			ret = append(ret, v.Copy())
		}
	}
	return ret
}

func GetContainerStatus(statuses []ContainerStatus, name string) (ContainerStatus, bool) {
	for i := range statuses {
		if statuses[i].Name == name {
			return statuses[i], true
		}
	}
	return ContainerStatus{}, false
}

func GetExistingContainerStatus(statuses []ContainerStatus, name string) ContainerStatus {
	for i := range statuses {
		if statuses[i].Name == name {
			return statuses[i]
		}
	}
	return ContainerStatus{}
}

// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// IsPodReady retruns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == ConditionTrue
}

// Extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetPodReadyCondition(status PodStatus) *PodCondition {
	_, condition := GetPodCondition(&status, PodReady)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetPodCondition(status *PodStatus, conditionType PodConditionType) (int, *PodCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// GetNodeCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetNodeCondition(status *NodeStatus, conditionType NodeConditionType) (int, *NodeCondition) {
	if status == nil {
		return -1, nil
	}
	for i := range status.Conditions {
		if status.Conditions[i].Type == conditionType {
			return i, &status.Conditions[i]
		}
	}
	return -1, nil
}

// Updates existing pod condition or creates a new one. Sets LastTransitionTime to now if the
// status has changed.
// Returns true if pod condition has changed or has been added.
func UpdatePodCondition(status *PodStatus, condition *PodCondition) bool {
	condition.LastTransitionTime = unversioned.Now()
	// Try to find this pod condition.
	conditionIndex, oldCondition := GetPodCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new pod condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	} else {
		// We are updating an existing condition, so we need to check if it has changed.
		if condition.Status == oldCondition.Status {
			condition.LastTransitionTime = oldCondition.LastTransitionTime
		}

		isEqual := condition.Status == oldCondition.Status &&
			condition.Reason == oldCondition.Reason &&
			condition.Message == oldCondition.Message &&
			condition.LastProbeTime.Equal(oldCondition.LastProbeTime) &&
			condition.LastTransitionTime.Equal(oldCondition.LastTransitionTime)

		status.Conditions[conditionIndex] = *condition
		// Return true if one of the fields have changed.
		return !isEqual
	}
}

// IsNodeReady returns true if a node is ready; false otherwise.
func IsNodeReady(node *Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == NodeReady {
			return c.Status == ConditionTrue
		}
	}
	return false
}

// PodRequestsAndLimits returns a dictionary of all defined resources summed up for all
// containers of the pod.
func PodRequestsAndLimits(pod *Pod) (reqs map[ResourceName]resource.Quantity, limits map[ResourceName]resource.Quantity, err error) {
	reqs, limits = map[ResourceName]resource.Quantity{}, map[ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			value, ok := reqs[name]
			if !ok {
				reqs[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = *quantity.Copy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = *quantity.Copy()
			}
		}
	}
	return
}
