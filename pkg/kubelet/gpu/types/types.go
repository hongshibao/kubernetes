package types

type GPUDeviceInfo struct {
	Name            string
	TotalMemory     int64
	AvailableMemory int64
}

type GPUProbe interface {
	GetGPUDeviceInfo() ([]GPUDeviceInfo, error)
}
