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
data "hypercore_vms" "nicvm" {
  name = %[1]q
}

resource "hypercore_nic" "test" {
  vm_uuid = data.hypercore_vms.nicvm.vms.0.uuid
  vlan    = 11
  type    = "VIRTIO"
}
`, source_vm_name)
}

func TestAccHypercoreNicResource_Mac(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercoreSourceVMRConfig_Mac(12, "INTEL_E1000", "52:54:00:11:22:33"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_nic.test_mac", "vlan", "12"),
					resource.TestCheckResourceAttr("hypercore_nic.test_mac", "type", "INTEL_E1000"),
					resource.TestCheckResourceAttr("hypercore_nic.test_mac", "mac_address", "52:54:00:11:22:33"),
				),
			},
			{
				Config: testAccHypercoreSourceVMRConfig_Mac(13, "RTL8139", "52:54:00:11:22:44"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hypercore_nic.test_mac", "vlan", "13"),
					resource.TestCheckResourceAttr("hypercore_nic.test_mac", "type", "RTL8139"),
					resource.TestCheckResourceAttr("hypercore_nic.test_mac", "mac_address", "52:54:00:11:22:44"),
				),
			},
		},
	})
}

func testAccHypercoreSourceVMRConfig_Mac(vlan int, nicType string, macAddress string) string {
	return fmt.Sprintf(`
data "hypercore_vms" "nicvm" {
  name = %[1]q
}

resource "hypercore_nic" "test_mac" {
  vm_uuid = data.hypercore_vms.nicvm.vms.0.uuid
  vlan    = %[2]d
  type    = %[3]q
  mac_address = %[4]q
}
`, source_vm_name, vlan, nicType, macAddress)
}
