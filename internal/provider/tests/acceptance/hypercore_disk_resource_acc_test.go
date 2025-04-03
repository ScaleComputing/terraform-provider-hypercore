// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHypercoreDiskResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Clone VM and create NIC
			{
				Config: testAccHypercoreDiskResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_disk.test", "size", "3.0"),
					resource.TestCheckResourceAttr("hypercore_disk.test", "type", "IDE_DISK"),
				),
			},
		},
	})
}

func testAccHypercoreDiskResourceConfig() string {
	return fmt.Sprintf(`
data "hypercore_vm" "diskvm" {
  name = %[1]q
}

resource "hypercore_disk" "test" {
  vm_uuid = data.hypercore_vm.diskvm.vms.0.uuid
  type    = "IDE_DISK"
  size    = 3.0
}

output "vm_id" {
  value = data.hypercore_vm.diskvm.vms.0.uuid
}
`, source_vm_name)
}
