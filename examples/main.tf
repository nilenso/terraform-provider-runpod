terraform {
  required_providers {
    runpod = {
      source = "nilenso/runpod"
    }
  }
}

# Configure the RunPod provider
# API key can be set via RUNPOD_API_KEY environment variable
provider "runpod" {
  # api_key = "your-api-key"  # Or use RUNPOD_API_KEY env var
}

# List available GPU types
data "runpod_gpu_types" "all" {
}

# Get a specific GPU type
data "runpod_gpu_types" "a4000" {
  filter {
    id = "NVIDIA RTX A4000"
  }
}

# Output available GPU types
output "gpu_types" {
  value = data.runpod_gpu_types.all.gpu_types
}

output "a4000_info" {
  value = data.runpod_gpu_types.a4000.gpu_types
}

# Create a basic GPU pod
resource "runpod_pod" "example" {
  name       = "my-terraform-pod"
  image_name = "runpod/pytorch:2.1.0-py3.10-cuda11.8.0-devel-ubuntu22.04"
  
  # You can specify a single GPU type
  # gpu_type_id = "NVIDIA RTX A6000"
  
  # Or a list of acceptable GPU types (RunPod will pick an available one)
  gpu_type_ids = ["NVIDIA RTX A4000", "NVIDIA RTX A5000"]
  
  gpu_count          = 1
  volume_in_gb       = 40
  container_disk_in_gb = 20
  
  # Optional settings
  cloud_type       = "ALL"      # ALL, SECURE, or COMMUNITY
  ports            = "8888/http,22/tcp"
  volume_mount_path = "/workspace"
  
  # Environment variables
  env = {
    JUPYTER_PASSWORD = "mysecretpassword"
    MY_CUSTOM_VAR    = "my_value"
  }
}

output "pod_id" {
  value = runpod_pod.example.id
}

output "pod_machine_id" {
  value = runpod_pod.example.machine_id
}
