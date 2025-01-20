// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScaleVMResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccScaleVMResourceConfig("tf-vm", "tf-src"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scale_vm.test", "description", ""), // TODO "tf-vm-description"
					resource.TestCheckResourceAttr("scale_vm.test", "memory", "0"),     // TODO 1 GB
					// resource.TestCheckResourceAttr("scale_vm.test", "nics", []),
					resource.TestCheckResourceAttr("scale_vm.test", "network_mode", ""),
					resource.TestCheckResourceAttr("scale_vm.test", "group", ""), // TODO "testtf"
					resource.TestCheckResourceAttr("scale_vm.test", "meta_data", ""),
					resource.TestCheckResourceAttr("scale_vm.test", "user_data", ""),
					resource.TestCheckResourceAttr("scale_vm.test", "network_iface", ""),
					resource.TestCheckResourceAttr("scale_vm.test", "source_vm_name", "tf-src"),
					resource.TestCheckResourceAttr("scale_vm.test", "name", "tf-vm"),
					resource.TestCheckResourceAttr("scale_vm.test", "vcpu", "0"),
					resource.TestCheckResourceAttr("scale_vm.test", "disk_size", "0"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "scale_vm.test",
				ImportState:       true,
				ImportStateVerify: true,
				// This is not normally necessary, but is here because this
				// example code does not have an actual upstream service.
				// Once the Read method is able to refresh information from
				// the upstream service, this can be removed.
				ImportStateVerifyIgnore: []string{
					"vm_list",
					// TODO do not ignore below attributes
					"name",
					"description",
					"group",
					"vcpu",
					"memory",
					"disk_size",
					"network_iface",
					"network_mode",
					"source_vm_name",
					"user_data",
					"meta_data",
				},
			},
			// Update and Read testing
			{
				Config: testAccScaleVMResourceConfig("tf-vm", "tf-src"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scale_vm.test", "name", "tf-vm"),
					resource.TestCheckResourceAttr("scale_vm.test", "description", ""),
					resource.TestCheckResourceAttr("scale_vm.test", "group", ""),
					resource.TestCheckResourceAttr("scale_vm.test", "vcpu", "0"),
					resource.TestCheckResourceAttr("scale_vm.test", "memory", "0"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccScaleVMResourceConfig(vm_name string, source_vm_name string) string {
	return fmt.Sprintf(`
resource "scale_vm" "test" {
  name = %[1]q
  source_vm_name = %[2]q
  group = ""
  vcpu = 0
  memory = 0
  network_mode = ""
  user_data = ""
  meta_data = ""
  network_iface = ""
  description = ""
  nics = []
  disk_size = 0
}
`, vm_name, source_vm_name)
}
