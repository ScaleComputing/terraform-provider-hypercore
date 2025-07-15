// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

/*
func testCheckListIsPresentEvenIfEmpty(resourceName string, attrName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		// val, ok := rs.Primary.Attributes[attrName+".#"]
		_, ok = rs.Primary.Attributes[attrName+".#"]
		if !ok {
			return fmt.Errorf("attribute %s is not set", attrName)
		}
		// val is a string representing the number of elements
		// if val != "0" {
		// 	return fmt.Errorf("expected %s to be empty, got length: %s", attrName, val)
		// }
		return nil
	}
}
*/

func TestAccHypercoreVMsDatasource_stopped(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testConfig_stopped("integration-test-vm"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.name", "integration-test-vm"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.memory", "4096"),
					// resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.vcpu", "1"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.power_state", "SHUTOFF"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.disks.#", "2"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.disks.0.type", "VIRTIO_DISK"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.disks.0.slot", "0"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.disks.0.size", "1.2"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.disks.1.type", "VIRTIO_DISK"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.disks.1.slot", "1"),
					// resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.disks.1.size", "2.4"),
				),
			},
		},
	})
}

func testConfig_stopped(vm_name string) string {
	return fmt.Sprintf(`
data "hypercore_vms" "test" {
  name = %[1]q
}
`, vm_name)
}

/*
TODO - test VM IP is actually returned.
To test this, we need a bootable ISO, with qemu-geust-agent.
Or similar qcow2 image.
func x_TestAccHypercoreVMsDatasource_running(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testConfig_running_step1(),
			},
			{
				Config: testConfig_running_step2(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.name", "testtf-datasource-vms-running"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.memory", "4096"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.vcpu", "4"),
					resource.TestCheckResourceAttr("data.hypercore_vms.test", "vms.0.power_state", "RUNNING"),
				),
			},
		},
	})
}

func testConfig_running_step1() string {
	return fmt.Sprintf(`
# bootable ISO, it needs to get IP from DHCP, and needs to contain qemu guest agent.
resource "hypercore_iso" "liveiso" {
  name       = "Porteus-XFCE-v5.0-x86_64.iso"
  source_url = "https://mirrors.dotsrc.org/porteus/x86_64/Porteus-v5.0/Porteus-XFCE-v5.0-x86_64.iso"
}

resource "hypercore_vm" "test" {
  tags        = ["testtf", "vms-datasource-running"]
  name        = "testtf-vms-datasource-running"
  description = "testtf-vms-datasource-running description"

  vcpu              = 4
  memory            = 4096 # MiB
  affinity_strategy = {}
}

resource "hypercore_disk" "liveiso" {
  vm_uuid  = hypercore_vm.test.id
  type     = "IDE_CDROM"
  iso_uuid = hypercore_iso.liveiso.id
  size     = 0.364904448
}
`)
}

func testConfig_running_step2() string {
	return fmt.Sprintf(`
# bootable ISO, it needs to get IP from DHCP, and needs to contain qemu guest agent.
resource "hypercore_iso" "liveiso" {
  name       = "Porteus-XFCE-v5.0-x86_64.iso"
  source_url = "https://mirrors.dotsrc.org/porteus/x86_64/Porteus-v5.0/Porteus-XFCE-v5.0-x86_64.iso"
}

resource "hypercore_vm" "test" {
  tags        = ["testtf", "vms-datasource-running"]
  name        = "testtf-vms-datasource-running"
  description = "testtf-vms-datasource-running description"

  vcpu              = 4
  memory            = 4096 # MiB
  affinity_strategy = {}
}

resource "hypercore_disk" "liveiso" {
  vm_uuid  = hypercore_vm.test.id
  type     = "IDE_CDROM"
  iso_uuid = hypercore_iso.liveiso.id
  size     = 0.364904448
}

data "hypercore_vms" "test" {
  name = "testtf-vms-datasource-running"
}
`)
}
*/
