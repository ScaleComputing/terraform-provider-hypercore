// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHypercoreVirtualDiskResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercoreVirtualDiskResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_virtual_disk.test", "size", "3"),
					resource.TestCheckResourceAttr("hypercore_virtual_disk.test", "type", "IDE_DISK"),
				),
			},
		},
	})
}

func testAccHypercoreVirtualDiskResourceConfig() string {
	return fmt.Sprintf(`
data "hypercore_vm" "integrationvm" {
  name = %[1]q
}

resource "hypercore_disk" "attach_vd" {
  vm_uuid                = data.hypercore_vm.integrationvm.vms.0.uuid
  type                   = "VIRTIO_DISK"
  size                   = 3.4
  source_virtual_disk_id = %[2]q
}

`, source_vm_name, existing_vdisk_uuid)
}
