// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHypercoreNicResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercoreSourceVMRConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_nic.test", "vlan", "11"),
					resource.TestCheckResourceAttr("hypercore_nic.test", "type", "VIRTIO"),
				),
			},
		},
	})
}

func testAccHypercoreSourceVMRConfig() string {
	return fmt.Sprintf(`
data "hypercore_vm" "nicvm" {
  name = %[1]q
}

resource "hypercore_nic" "test" {
  vm_uuid = data.hypercore_vm.nicvm.vms.0.uuid
  vlan    = 11
  type    = "VIRTIO"
}

output "vm_id" {
  value = data.hypercore_vm.nicvm.vms.0.uuid
}
`, source_vm_name)
}
