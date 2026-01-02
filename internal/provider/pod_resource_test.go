package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPodResource_lifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccPodResourceConfig("tf-test-pod", 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("runpod_pod.test", "name", "tf-test-pod"),
					resource.TestCheckResourceAttr("runpod_pod.test", "volume_in_gb", "20"),
					resource.TestCheckResourceAttr("runpod_pod.test", "gpu_count", "1"),
					resource.TestCheckResourceAttrSet("runpod_pod.test", "id"),
				),
			},
			// Update volume size
			{
				Config: testAccPodResourceConfig("tf-test-pod", 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("runpod_pod.test", "volume_in_gb", "30"),
				),
			},
			// Import
			{
				ResourceName:            "runpod_pod.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"gpu_type_ids", "cloud_type", "env", "support_public_ip", "start_ssh", "min_vcpu_count", "min_memory_in_gb"},
			},
			// Delete happens automatically
		},
	})
}

func testAccPodResourceConfig(name string, volumeGb int) string {
	return fmt.Sprintf(`
resource "runpod_pod" "test" {
  name               = %[1]q
  image_name         = "runpod/pytorch:2.1.0-py3.10-cuda11.8.0-devel-ubuntu22.04"
  gpu_type_ids       = ["NVIDIA RTX A4000"]
  gpu_count          = 1
  volume_in_gb       = %[2]d
  container_disk_in_gb = 20
}
`, name, volumeGb)
}

func TestAccPodResource_withEnv(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPodResourceConfigWithEnv(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("runpod_pod.test_env", "name", "tf-test-pod-env"),
					resource.TestCheckResourceAttr("runpod_pod.test_env", "env.TEST_VAR", "test_value"),
					resource.TestCheckResourceAttrSet("runpod_pod.test_env", "id"),
				),
			},
		},
	})
}

func testAccPodResourceConfigWithEnv() string {
	return `
resource "runpod_pod" "test_env" {
  name               = "tf-test-pod-env"
  image_name         = "runpod/pytorch:2.1.0-py3.10-cuda11.8.0-devel-ubuntu22.04"
  gpu_type_ids       = ["NVIDIA RTX A4000"]
  gpu_count          = 1
  volume_in_gb       = 20
  container_disk_in_gb = 20
  
  env = {
    TEST_VAR = "test_value"
    ANOTHER_VAR = "another_value"
  }
}
`
}
