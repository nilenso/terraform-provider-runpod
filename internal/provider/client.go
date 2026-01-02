package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://api.runpod.io/graphql"

// Client handles communication with the RunPod GraphQL API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	mu         sync.Mutex // ensures sequential API calls
}

// NewClient creates a new RunPod API client
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GraphQL request/response types
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

func (c *Client) doRequest(query string, variables map[string]interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Retry with exponential backoff for rate limiting
	maxRetries := 5
	baseDelay := 2 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		url := fmt.Sprintf("%s?api_key=%s", c.baseURL, c.apiKey)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Retry on 429 Too Many Requests or 503 Service Unavailable
		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<attempt)
				time.Sleep(delay)
				continue
			}
		}

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
		}

		var gqlResp graphQLResponse
		if err := json.Unmarshal(respBody, &gqlResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if len(gqlResp.Errors) > 0 {
			return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
		}

		return gqlResp.Data, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// Ping tests the API connection by querying the current user
func (c *Client) Ping() error {
	query := `query { myself { id } }`
	_, err := c.doRequest(query, nil)
	return err
}

// Pod represents a RunPod pod
type Pod struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	ImageName         string   `json:"imageName"`
	GpuTypeID         string   `json:"gpuTypeId"`
	GpuCount          int      `json:"gpuCount"`
	VolumeInGb        int      `json:"volumeInGb"`
	ContainerDiskInGb int      `json:"containerDiskInGb"`
	DesiredStatus     string   `json:"desiredStatus"`
	CloudType         string   `json:"cloudType"`
	Ports             string   `json:"ports"`
	VolumeMountPath   string   `json:"volumeMountPath"`
	DockerArgs        string   `json:"dockerArgs"`
	Env               EnvVars  `json:"env"`
	MachineID         string   `json:"machineId"`
	Machine           *Machine `json:"machine"`
	Runtime           *Runtime `json:"runtime"`
}

type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// EnvVars is a slice of EnvVar that handles custom JSON unmarshalling
// The API returns env as string array like ["KEY=value"] but we want []EnvVar
type EnvVars []EnvVar

func (e *EnvVars) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string array first (API response format)
	var stringArray []string
	if err := json.Unmarshal(data, &stringArray); err == nil {
		*e = make(EnvVars, 0, len(stringArray))
		for _, s := range stringArray {
			parts := strings.SplitN(s, "=", 2)
			if len(parts) == 2 {
				*e = append(*e, EnvVar{Key: parts[0], Value: parts[1]})
			} else if len(parts) == 1 {
				*e = append(*e, EnvVar{Key: parts[0], Value: ""})
			}
		}
		return nil
	}

	// Fall back to unmarshalling as []EnvVar (for other cases)
	var envVars []EnvVar
	if err := json.Unmarshal(data, &envVars); err != nil {
		return err
	}
	*e = envVars
	return nil
}

type Machine struct {
	PodHostID string `json:"podHostId"`
}

type Runtime struct {
	UptimeInSeconds int     `json:"uptimeInSeconds"`
	Ports           []Port  `json:"ports"`
}

type Port struct {
	IP          string `json:"ip"`
	IsIPPublic  bool   `json:"isIpPublic"`
	PrivatePort int    `json:"privatePort"`
	PublicPort  int    `json:"publicPort"`
	Type        string `json:"type"`
}

// PodInput represents the input for creating a pod
type PodInput struct {
	Name              string   `json:"name"`
	ImageName         string   `json:"imageName"`
	GpuTypeID         string   `json:"gpuTypeId,omitempty"`
	GpuTypeIDs        []string `json:"gpuTypeIds,omitempty"`
	GpuCount          int      `json:"gpuCount"`
	VolumeInGb        int      `json:"volumeInGb"`
	ContainerDiskInGb int      `json:"containerDiskInGb"`
	CloudType         string   `json:"cloudType,omitempty"`
	Ports             string   `json:"ports,omitempty"`
	VolumeMountPath   string   `json:"volumeMountPath,omitempty"`
	DockerArgs        string   `json:"dockerArgs,omitempty"`
	Env               []EnvVar `json:"env,omitempty"`
	MinVcpuCount      int      `json:"minVcpuCount,omitempty"`
	MinMemoryInGb     int      `json:"minMemoryInGb,omitempty"`
	NetworkVolumeID   string   `json:"networkVolumeId,omitempty"`
	TemplateID        string   `json:"templateId,omitempty"`
	DataCenterID      string   `json:"dataCenterId,omitempty"`
	SupportPublicIP   bool     `json:"supportPublicIp,omitempty"`
	StartSSH          bool     `json:"startSsh,omitempty"`
}

// CreatePod creates a new on-demand pod
func (c *Client) CreatePod(input *PodInput) (*Pod, error) {
	query := `mutation PodFindAndDeployOnDemand($input: PodFindAndDeployOnDemandInput!) {
		podFindAndDeployOnDemand(input: $input) {
			id
			name
			imageName
			gpuCount
			volumeInGb
			containerDiskInGb
			desiredStatus
			ports
			volumeMountPath
			dockerArgs
			env
			machineId
			machine {
				podHostId
			}
		}
	}`

	// Build the input map for the GraphQL query
	inputMap := map[string]interface{}{
		"name":              input.Name,
		"imageName":         input.ImageName,
		"gpuCount":          input.GpuCount,
		"volumeInGb":        input.VolumeInGb,
		"containerDiskInGb": input.ContainerDiskInGb,
	}

	// Handle GPU type - API expects gpuTypeId (singular string)
	// If gpuTypeIDs is provided, use the first one
	if len(input.GpuTypeIDs) > 0 {
		inputMap["gpuTypeId"] = input.GpuTypeIDs[0]
	} else if input.GpuTypeID != "" {
		inputMap["gpuTypeId"] = input.GpuTypeID
	}

	if input.CloudType != "" {
		inputMap["cloudType"] = input.CloudType
	}
	if input.Ports != "" {
		inputMap["ports"] = input.Ports
	}
	if input.VolumeMountPath != "" {
		inputMap["volumeMountPath"] = input.VolumeMountPath
	}
	if input.DockerArgs != "" {
		inputMap["dockerArgs"] = input.DockerArgs
	}
	if len(input.Env) > 0 {
		envList := make([]map[string]string, len(input.Env))
		for i, e := range input.Env {
			envList[i] = map[string]string{"key": e.Key, "value": e.Value}
		}
		inputMap["env"] = envList
	}
	if input.MinVcpuCount > 0 {
		inputMap["minVcpuCount"] = input.MinVcpuCount
	}
	if input.MinMemoryInGb > 0 {
		inputMap["minMemoryInGb"] = input.MinMemoryInGb
	}
	if input.NetworkVolumeID != "" {
		inputMap["networkVolumeId"] = input.NetworkVolumeID
	}
	if input.TemplateID != "" {
		inputMap["templateId"] = input.TemplateID
	}
	if input.DataCenterID != "" {
		inputMap["dataCenterId"] = input.DataCenterID
	}
	if input.SupportPublicIP {
		inputMap["supportPublicIp"] = input.SupportPublicIP
	}
	if input.StartSSH {
		inputMap["startSsh"] = input.StartSSH
	}

	variables := map[string]interface{}{
		"input": inputMap,
	}

	data, err := c.doRequest(query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %w", err)
	}

	var result struct {
		PodFindAndDeployOnDemand *Pod `json:"podFindAndDeployOnDemand"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod response: %w", err)
	}

	if result.PodFindAndDeployOnDemand == nil {
		return nil, fmt.Errorf("no pod returned from API")
	}

	return result.PodFindAndDeployOnDemand, nil
}

// GetPod retrieves a pod by ID
func (c *Client) GetPod(id string) (*Pod, error) {
	query := `query Pod($input: PodFilter!) {
		pod(input: $input) {
			id
			name
			imageName
			gpuCount
			volumeInGb
			containerDiskInGb
			desiredStatus
			ports
			volumeMountPath
			dockerArgs
			env
			machineId
			machine {
				podHostId
			}
			runtime {
				uptimeInSeconds
				ports {
					ip
					isIpPublic
					privatePort
					publicPort
					type
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"input": map[string]string{
			"podId": id,
		},
	}

	data, err := c.doRequest(query, variables)
	if err != nil {
		return nil, err
	}

	var result struct {
		Pod *Pod `json:"pod"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod response: %w", err)
	}

	if result.Pod == nil {
		return nil, fmt.Errorf("pod not found: %s", id)
	}

	return result.Pod, nil
}

// TerminatePod terminates (deletes) a pod
func (c *Client) TerminatePod(id string) error {
	query := `mutation PodTerminate($input: PodTerminateInput!) {
		podTerminate(input: $input)
	}`

	variables := map[string]interface{}{
		"input": map[string]string{
			"podId": id,
		},
	}

	_, err := c.doRequest(query, variables)
	if err != nil {
		return fmt.Errorf("failed to terminate pod: %w", err)
	}

	return nil
}

// StopPod stops a pod (without terminating it)
func (c *Client) StopPod(id string) (*Pod, error) {
	query := `mutation PodStop($input: PodStopInput!) {
		podStop(input: $input) {
			id
			desiredStatus
		}
	}`

	variables := map[string]interface{}{
		"input": map[string]string{
			"podId": id,
		},
	}

	data, err := c.doRequest(query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to stop pod: %w", err)
	}

	var result struct {
		PodStop *Pod `json:"podStop"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod response: %w", err)
	}

	return result.PodStop, nil
}

// ResumePod resumes/starts a stopped pod
func (c *Client) ResumePod(id string, gpuCount int) (*Pod, error) {
	query := `mutation PodResume($input: PodResumeInput!) {
		podResume(input: $input) {
			id
			desiredStatus
			imageName
			machineId
			machine {
				podHostId
			}
		}
	}`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"podId":    id,
			"gpuCount": gpuCount,
		},
	}

	data, err := c.doRequest(query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to resume pod: %w", err)
	}

	var result struct {
		PodResume *Pod `json:"podResume"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod response: %w", err)
	}

	return result.PodResume, nil
}

// GpuType represents a GPU type available on RunPod
type GpuType struct {
	ID             string  `json:"id"`
	DisplayName    string  `json:"displayName"`
	MemoryInGb     int     `json:"memoryInGb"`
	SecureCloud    bool    `json:"secureCloud"`
	CommunityCloud bool    `json:"communityCloud"`
}

// ListGpuTypes retrieves all available GPU types
func (c *Client) ListGpuTypes() ([]GpuType, error) {
	query := `query GpuTypes {
		gpuTypes {
			id
			displayName
			memoryInGb
			secureCloud
			communityCloud
		}
	}`

	data, err := c.doRequest(query, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		GpuTypes []GpuType `json:"gpuTypes"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gpu types response: %w", err)
	}

	return result.GpuTypes, nil
}

// GetGpuType retrieves a specific GPU type by ID
func (c *Client) GetGpuType(id string) (*GpuType, error) {
	query := `query GpuTypes {
		gpuTypes(input: {id: "` + id + `"}) {
			id
			displayName
			memoryInGb
			secureCloud
			communityCloud
		}
	}`

	variables := map[string]interface{}{}

	data, err := c.doRequest(query, variables)
	if err != nil {
		return nil, err
	}

	var result struct {
		GpuTypes []GpuType `json:"gpuTypes"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gpu type response: %w", err)
	}

	if len(result.GpuTypes) == 0 {
		return nil, fmt.Errorf("GPU type not found: %s", id)
	}

	return &result.GpuTypes[0], nil
}
