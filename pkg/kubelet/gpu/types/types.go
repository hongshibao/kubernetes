package types

type GPUDeviceInfo struct {
	Name string
	Path string
	// in MB
	TotalMemory     int64
	AvailableMemory int64
}

type GPUProbe interface {
	GetGPUDeviceInfo() ([]GPUDeviceInfo, error)
}
