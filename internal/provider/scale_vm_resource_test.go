// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var requested_power_state string = "stop" // "started"
// UUID of VM with name "testtf_src"
// var testtf_src_uuid string = "27af8248-88ee-4420-85d7-78b735415064"  // https://172.31.6.11
var testtf_src_uuid string = "ff36479e-06bb-4141-bad5-0097c8c1a4a6" // https://10.5.11.205

func TestAccScaleVMResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccScaleVMResourceConfig("testtf-vm", testtf_src_uuid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scale_vm.test", "description", "testtf-vm-description"),
					resource.TestCheckResourceAttr("scale_vm.test", "memory", "4096"),
					resource.TestCheckResourceAttr("scale_vm.test", "group", "testtf"),
					resource.TestCheckResourceAttr("scale_vm.test", "clone.meta_data", ""),
					resource.TestCheckResourceAttr("scale_vm.test", "clone.user_data", ""),
					resource.TestCheckResourceAttr("scale_vm.test", "clone.source_vm_uuid", testtf_src_uuid),
					resource.TestCheckResourceAttr("scale_vm.test", "name", "testtf-vm"),
					resource.TestCheckResourceAttr("scale_vm.test", "vcpu", "4"),
					// resource.TestCheckResourceAttr("scale_vm.test", "power_state", requested_power_state),
				),
			},
			// TODO make ImportState test pass again.
			/*
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
						"id",
						// TODO do not ignore below attributes
						"name",
						"description",
						"group",
						"vcpu",
						"memory",
						"disk_size",
						"clone.source_vm_uuid",
						"clone.user_data",
						"clone.meta_data",
						"power_state",
					},
				},
			*/
			// Update and Read testing
			{
				Config: testAccScaleVMResourceConfig("testtf-vm", testtf_src_uuid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scale_vm.test", "name", "testtf-vm"),
					resource.TestCheckResourceAttr("scale_vm.test", "description", "testtf-vm-description"),
					resource.TestCheckResourceAttr("scale_vm.test", "group", "testtf"),
					resource.TestCheckResourceAttr("scale_vm.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("scale_vm.test", "memory", "4096"),
					// resource.TestCheckResourceAttr("scale_vm.test", "power_state", requested_power_state),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccScaleVMResourceConfig(vm_name string, source_vm_uuid string) string {
	return fmt.Sprintf(`
resource "scale_vm" "test" {
  name = %[1]q
  group = "testtf"
  vcpu = 4
  memory = 4096
  description = "testtf-vm-description"
  // power_state = %[3]q
  clone = {
	source_vm_uuid = %[2]q
	user_data = ""
	meta_data = ""
  }
}
`, vm_name, source_vm_uuid, requested_power_state)
}
