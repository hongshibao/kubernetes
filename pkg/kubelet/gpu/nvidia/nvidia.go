package nvidia

import (
	gputypes "k8s.io/kubernetes/pkg/kubelet/gpu/types"
)

type NvidiaGPU struct {
}

func NewNvidiaGPU() (*NvidiaGPU, error) {
	return &NvidiaGPU{}, nil
}

func (self *NvidiaGPU) GetGPUDeviceInfo() ([]gputypes.GPUDeviceInfo, error) {
	return []gputypes.GPUDeviceInfo{
		gputypes.GPUDeviceInfo{
			Name:            "gpu0",
			TotalMemory:     4096,
			AvailableMemory: 3500,
		},
	}, nil
}
