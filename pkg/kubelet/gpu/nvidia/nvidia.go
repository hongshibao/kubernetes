package nvidia

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	gputypes "k8s.io/kubernetes/pkg/kubelet/gpu/types"
)

const gpuRequestBufferTime = 10 * time.Second

type staticGPUInfo struct {
	Name        string
	Path        string
	TotalMemory int64
}

type gpuRequest struct {
	req       int64
	timeStamp time.Time
}

type gpuRequestBuffer struct {
	gpuRequests []gpuRequest
	sumRequest  int64
}

type NvidiaGPU struct {
	staticGPUInfo     []staticGPUInfo
	bufferMutex       sync.Mutex
	gpuRequestBuffers []gpuRequestBuffer
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

func getStaticGPUInfo() ([]staticGPUInfo, error) {
	resp, err := getHttpResponse("http://127.0.0.1:3476/v1.0/gpu/info/json")
	if err != nil {
		return nil, fmt.Errorf("Error getting http response: %s", err)
	}

	gpuInfoMap := make(map[string]interface{})
	err = json.Unmarshal(resp, &gpuInfoMap)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling GPU info data: %s", err)
	}
	staticGPUInfoArray := make([]staticGPUInfo, 0)
	gpuInfoArr := gpuInfoMap["Devices"].([]interface{})
	for _, gpuInfo := range gpuInfoArr {
		info := gpuInfo.(map[string]interface{})
		gpu := staticGPUInfo{
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
		staticGPUInfoArray = append(staticGPUInfoArray, gpu)
	}

	return staticGPUInfoArray, nil
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

func (self *NvidiaGPU) GetBufferedGPUDeviceInfo() ([]gputypes.GPUDeviceInfo, error) {
	currentTime := time.Now()
	gpuInfo, err := self.GetGPUDeviceInfo()
	if err != nil {
		return nil, err
	}

	self.bufferMutex.Lock()
	defer self.bufferMutex.Unlock()
	if self.gpuRequestBuffers == nil {
		self.gpuRequestBuffers = make([]gpuRequestBuffer, len(gpuInfo))
	}
	cutTimeline := currentTime.Add(-gpuRequestBufferTime)
	for i := range self.gpuRequestBuffers {
		for j := 0; j < len(self.gpuRequestBuffers[i].gpuRequests); j++ {
			if self.gpuRequestBuffers[i].gpuRequests[j].timeStamp.Before(cutTimeline) {
				self.gpuRequestBuffers[i].sumRequest -= self.gpuRequestBuffers[i].gpuRequests[j].req
				lastIndex := len(self.gpuRequestBuffers[i].gpuRequests) - 1
				self.gpuRequestBuffers[i].gpuRequests[j] = self.gpuRequestBuffers[i].gpuRequests[lastIndex]
				self.gpuRequestBuffers[i].gpuRequests = self.gpuRequestBuffers[i].gpuRequests[:lastIndex]
				j--
			}
		}
	}
	for i := range gpuInfo {
		gpuInfo[i].AvailableMemory -= self.gpuRequestBuffers[i].sumRequest
		if gpuInfo[i].AvailableMemory < 0 {
			gpuInfo[i].AvailableMemory = 0
		}
	}
	return gpuInfo, nil
}

func (self *NvidiaGPU) AddGPURequest(idx int, request int64) error {
	self.bufferMutex.Lock()
	defer self.bufferMutex.Unlock()
	if idx >= len(self.gpuRequestBuffers) {
		return fmt.Errorf("Index %d out of range: max value %d", idx, len(self.gpuRequestBuffers))
	}
	self.gpuRequestBuffers[idx].gpuRequests = append(self.gpuRequestBuffers[idx].gpuRequests, gpuRequest{
		req:       request,
		timeStamp: time.Now(),
	})
	self.gpuRequestBuffers[idx].sumRequest += request
	return nil
}
