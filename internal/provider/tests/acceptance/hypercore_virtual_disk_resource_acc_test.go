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
					resource.TestCheckResourceAttr("hypercore_disk.attach_vd_test", "size", "3.4"),
					resource.TestCheckResourceAttr("hypercore_disk.attach_vd_test", "type", "VIRTIO_DISK"),
					resource.TestCheckResourceAttr("hypercore_disk.attach_vd_test", "flash_priority", "4"),
				),
			},
		},
	})
}

func testAccHypercoreVirtualDiskResourceConfig() string {
	return fmt.Sprintf(`
data "hypercore_vms" "integrationvm" {
  name = %[1]q
}

resource "hypercore_disk" "attach_vd_test" {
  vm_uuid                = data.hypercore_vms.integrationvm.vms.0.uuid
  type                   = "VIRTIO_DISK"
  size                   = 3.4
  source_virtual_disk_id = %[2]q
}

`, source_vm_name, existing_vdisk_uuid)
}
