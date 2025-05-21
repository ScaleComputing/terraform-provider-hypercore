// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHypercoreVMResourceImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccHypercoreVMResourceImportConfig("imported_vm_integration"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_vm.test", "description", "imported-vm"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "memory", "4096"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.#", "1"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "tags.0", "Xlabintegrationtest"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "name", "imported_vm_integration"),
					resource.TestCheckResourceAttr("hypercore_vm.test", "vcpu", "4"),
				),
			},
		},
	})
}

func testAccHypercoreVMResourceImportConfig(vm_name string) string {
	return fmt.Sprintf(`
resource "hypercore_vm" "test" {
  name = %[1]q
  tags = ["Xlabintegrationtest"]
  vcpu = 4
  memory = 4096
  description = "imported-vm"
  snapshot_schedule_uuid = ""
  // power_state = %[3]q
  import = {
    server    = %[2]q
    username  = %[3]q
    password  = %[4]q
    path      = %[5]q
    file_name = %[6]q
  }
  affinity_strategy = {}
}
`, vm_name, smb_server, smb_username, smb_password, smb_path, smb_filename, requested_power_state)
}
