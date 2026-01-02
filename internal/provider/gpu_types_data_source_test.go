package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGpuTypesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGpuTypesDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.runpod_gpu_types.all", "id", "gpu_types"),
					resource.TestCheckResourceAttrSet("data.runpod_gpu_types.all", "gpu_types.#"),
				),
			},
		},
	})
}

func testAccGpuTypesDataSourceConfig() string {
	return `
data "runpod_gpu_types" "all" {
}
`
}

func TestAccGpuTypesDataSource_filtered(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGpuTypesDataSourceConfigFiltered(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.runpod_gpu_types.filtered", "gpu_types.#", "1"),
					resource.TestCheckResourceAttr("data.runpod_gpu_types.filtered", "gpu_types.0.id", "NVIDIA RTX A4000"),
				),
			},
		},
	})
}

func testAccGpuTypesDataSourceConfigFiltered() string {
	return `
data "runpod_gpu_types" "filtered" {
  filter {
    id = "NVIDIA RTX A4000"
  }
}
`
}
