# Terraform Provider for RunPod

[![Tests](https://github.com/nilenso/terraform-provider-runpod/actions/workflows/test.yml/badge.svg)](https://github.com/nilenso/terraform-provider-runpod/actions/workflows/test.yml)
[![Release](https://github.com/nilenso/terraform-provider-runpod/actions/workflows/release.yml/badge.svg)](https://github.com/nilenso/terraform-provider-runpod/actions/workflows/release.yml)

A Terraform provider for managing [RunPod](https://www.runpod.io/) GPU cloud resources.

## Features

- **Pod Management**: Create, update, and delete GPU pods
- **GPU Type Discovery**: Query available GPU types and their specifications
- **GPU Type Selection**: Specify the GPU type for your pod

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building from source)
- A RunPod account and API key

## Installation

### From Source

```bash
git clone https://github.com/nilenso/terraform-provider-runpod.git
cd terraform-provider-runpod
make install
```

### From Terraform Registry

Once published to the Terraform Registry, add to your `required_providers`:

```hcl
terraform {
  required_providers {
    runpod = {
      source  = "nilenso/runpod"
      version = "~> 0.1"
    }
  }
}
```

## Configuration

### Provider Configuration

```hcl
provider "runpod" {
  api_key = "your-api-key"  # Or use RUNPOD_API_KEY environment variable
}
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `RUNPOD_API_KEY` | Your RunPod API key |

## Usage

### Query Available GPU Types

```hcl
# List all available GPU types
data "runpod_gpu_types" "all" {
}

# Get a specific GPU type
data "runpod_gpu_types" "a4000" {
  filter {
    id = "NVIDIA RTX A4000"
  }
}

output "gpu_types" {
  value = data.runpod_gpu_types.all.gpu_types
}
```

### Create a GPU Pod

```hcl
resource "runpod_pod" "example" {
  name       = "my-gpu-pod"
  image_name = "runpod/pytorch:2.1.0-py3.10-cuda11.8.0-devel-ubuntu22.04"
  
  gpu_type_id        = "NVIDIA RTX A4000"
  gpu_count          = 1
  volume_in_gb       = 40
  container_disk_in_gb = 20
  
  # Optional settings
  cloud_type        = "ALL"  # ALL, SECURE, or COMMUNITY
  ports             = "8888/http,22/tcp"
  volume_mount_path = "/workspace"
  
  # Environment variables
  env = {
    JUPYTER_PASSWORD = "mysecretpassword"
  }
}

output "pod_id" {
  value = runpod_pod.example.id
}
```

## Resources

### runpod_pod

Manages a RunPod GPU pod.

#### Arguments

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | The name of the pod |
| `image_name` | string | Yes | Docker image to use |
| `gpu_type_id` | string | Yes | GPU type ID (e.g., "NVIDIA RTX A4000") |
| `gpu_count` | number | No | Number of GPUs (default: 1) |
| `volume_in_gb` | number | No | Persistent volume size in GB (default: 0) |
| `container_disk_in_gb` | number | No | Container disk size in GB (default: 20) |
| `cloud_type` | string | No | Cloud type: ALL, SECURE, COMMUNITY (default: ALL) |
| `ports` | string | No | Ports to expose (e.g., "8888/http,22/tcp") |
| `volume_mount_path` | string | No | Volume mount path (default: /workspace) |
| `docker_args` | string | No | Docker arguments |
| `env` | map(string) | No | Environment variables |
| `min_vcpu_count` | number | No | Minimum vCPUs required |
| `min_memory_in_gb` | number | No | Minimum memory in GB |
| `network_volume_id` | string | No | Network volume to attach |
| `template_id` | string | No | Template to use |
| `data_center_id` | string | No | Specific data center |
| `support_public_ip` | bool | No | Support public IP (default: true) |
| `start_ssh` | bool | No | Start SSH service (default: true) |

#### Attributes (Read-Only)

| Attribute | Description |
|-----------|-------------|
| `id` | The pod's unique identifier |
| `machine_id` | The machine ID the pod is running on |
| `pod_host_id` | The host ID of the pod |

#### Import

Pods can be imported using their ID:

```bash
terraform import runpod_pod.example <pod-id>
```

## Data Sources

### runpod_gpu_types

Fetches available GPU types from RunPod.

#### Arguments

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `filter.id` | string | No | Filter by GPU type ID |

#### Attributes

| Attribute | Description |
|-----------|-------------|
| `gpu_types` | List of GPU types |
| `gpu_types[].id` | GPU type ID (e.g., "NVIDIA RTX A6000") |
| `gpu_types[].display_name` | Display name |
| `gpu_types[].memory_in_gb` | GPU memory in GB |
| `gpu_types[].secure_cloud` | Available on secure cloud |
| `gpu_types[].community_cloud` | Available on community cloud |

## Development

### Building

```bash
make build
```

### Testing

```bash
# Unit tests
make test

# Acceptance tests (requires RUNPOD_API_KEY)
export RUNPOD_API_KEY="your-api-key"
make testacc

# Run a specific acceptance test
make testacc-one TEST=TestAccPodResource_lifecycle
```

### Installing Locally

```bash
make install
```

### Formatting and Linting

```bash
make fmt
make lint
```

## License

MIT License - see [LICENSE](LICENSE) for details.
