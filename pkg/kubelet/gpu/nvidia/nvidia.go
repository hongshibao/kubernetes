package nvidia

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	gputypes "k8s.io/kubernetes/pkg/kubelet/gpu/types"
)

type StaticGPUInfo struct {
	Name        string
	Path        string
	TotalMemory int64
}

type NvidiaGPU struct {
	staticGPUInfo []StaticGPUInfo
}

func NewNvidiaGPU() (*NvidiaGPU, error) {
	staticGPUInfo, err := getStaticGPUInfo()
	if err != nil {
		return nil, fmt.Errorf("Error getting static GPU info: %s", err)
	}
	return &NvidiaGPU{
		staticGPUInfo: staticGPUInfo,
	}, nil
}

func getHttpResponse(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Error in http Get: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading all body of http response: %s", err)
	}
	return body, nil
}

func getStaticGPUInfo() ([]StaticGPUInfo, error) {
	resp, err := getHttpResponse("http://127.0.0.1:3476/v1.0/gpu/info/json")
	if err != nil {
		return nil, fmt.Errorf("Error getting http response: %s", err)
	}

	gpuInfoMap := make(map[string]interface{})
	err = json.Unmarshal(resp, &gpuInfoMap)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling GPU info data: %s", err)
	}
	staticGPUInfo := make([]StaticGPUInfo, 0)
	gpuInfoArr := gpuInfoMap["Devices"].([]interface{})
	for _, gpuInfo := range gpuInfoArr {
		info := gpuInfo.(map[string]interface{})
		gpu := StaticGPUInfo{
			Name: info["UUID"].(string),
			Path: info["Path"].(string),
		}
		memoryInfo := info["Memory"].(map[string]interface{})
		//memory, err := strconv.ParseInt(memoryInfo["Global"].(string), 10, 64)
		memory := int64(memoryInfo["Global"].(float64))
		if err != nil {
			memory = 0
		}
		gpu.TotalMemory = memory
		staticGPUInfo = append(staticGPUInfo, gpu)
	}

	return staticGPUInfo, nil
}

func (self *NvidiaGPU) GetGPUDeviceInfo() ([]gputypes.GPUDeviceInfo, error) {
	resp, err := getHttpResponse("http://127.0.0.1:3476/v1.0/gpu/status/json")
	if err != nil {
		return nil, fmt.Errorf("Error getting http response: %s", err)
	}

	gpuStatusMap := make(map[string]interface{})
	err = json.Unmarshal(resp, &gpuStatusMap)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling GPU status data: %s", err)
	}

	gpuDeviceInfo := make([]gputypes.GPUDeviceInfo, 0)
	gpuStatusArr := gpuStatusMap["Devices"].([]interface{})
	for i, gpuStatus := range gpuStatusArr {
		status := gpuStatus.(map[string]interface{})
		memoryStatus := status["Memory"].(map[string]interface{})
		//memory, err := strconv.ParseInt(memoryStatus["GlobalUsed"].(string), 10, 64)
		memory := int64(memoryStatus["GlobalUsed"].(float64))
		if err != nil {
			memory = self.staticGPUInfo[i].TotalMemory
		}
		gpu := gputypes.GPUDeviceInfo{
			Name:            self.staticGPUInfo[i].Name,
			Path:            self.staticGPUInfo[i].Path,
			TotalMemory:     self.staticGPUInfo[i].TotalMemory,
			AvailableMemory: self.staticGPUInfo[i].TotalMemory - memory,
		}
		gpuDeviceInfo = append(gpuDeviceInfo, gpu)
	}

	return gpuDeviceInfo, nil
}
