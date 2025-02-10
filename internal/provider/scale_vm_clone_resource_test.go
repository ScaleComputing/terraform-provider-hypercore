// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var requested_power_state string = "stop" // "started"

func TestAccScaleVMCloneResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccScaleVMCloneResourceConfig("testtf-vm", "testtf-src-uuid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scale_vm_clone.test", "description", "testtf-vm-description"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "memory", "4096"),
					// resource.TestCheckResourceAttr("scale_vm_clone.test", "nics", []),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "group", "testtf"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "clone.meta_data", ""),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "clone.user_data", ""),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "clone.source_vm_uuid", "testtf-src-uuid"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "name", "testtf-vm"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "disk_size", "4"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "power_state", requested_power_state),
				),
			},
			// TODO make ImportState test pass again.
			/*
				// ImportState testing
				{
					ResourceName:      "scale_vm_clone.test",
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
				Config: testAccScaleVMCloneResourceConfig("testtf-vm", "testtf-src-uuid"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("scale_vm_clone.test", "name", "testtf-vm"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "description", "testtf-vm-description"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "group", "testtf"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "vcpu", "4"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "disk_size", "4"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "memory", "4096"),
					resource.TestCheckResourceAttr("scale_vm_clone.test", "power_state", requested_power_state),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccScaleVMCloneResourceConfig(vm_name string, source_vm_uuid string) string {
	return fmt.Sprintf(`
resource "scale_vm_clone" "test" {
  name = %[1]q
  group = "testtf"
  vcpu = 4
  memory = 4096
  description = "testtf-vm-description"
  nics = []
  disk_size = 4
  power_state = %[3]q
  clone = {
	source_vm_uuid = %[2]q
	user_data = ""
	meta_data = ""
  }
}
`, vm_name, source_vm_uuid, requested_power_state)
}
