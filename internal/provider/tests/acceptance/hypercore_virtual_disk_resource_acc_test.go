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

resource "hypercore_virtual_disk" "vd_upload_from_url" {
  name       = "virtual-disk-acc-test.img"
  source_url = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
}


`, source_vm_name)
}
