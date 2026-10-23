// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acceptance

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-hypercore/internal/utils"
)

// A VM deleted outside Terraform must be dropped from state on refresh, so the plan
// recreates it instead of failing with "VM not found" (issue #112).
// Self-contained: creates its own VM, needs no pre-existing fixtures.
func TestAccHypercoreVMResource_RemovedOutsideTerraform(t *testing.T) {
	vmName := "testtf-vm-removed-outside"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercoreVMResourceRemovedConfig(vmName),
				Check:  resource.TestCheckResourceAttr("hypercore_vm.test", "name", vmName),
			},
			{
				// Delete the VM behind Terraform's back, then expect a plan to create it.
				PreConfig:          func() { deleteVMByName(t, vmName) },
				Config:             testAccHypercoreVMResourceRemovedConfig(vmName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func deleteVMByName(t *testing.T, name string) {
	restClient := SetupRestClient()
	vms := utils.GetVM(map[string]any{"name": name}, *restClient)
	if len(vms) != 1 {
		t.Fatalf("expected exactly one VM named %s, got %d", name, len(vms))
	}
	uuid := utils.AnyToString(vms[0]["uuid"])
	ctx := context.Background()
	taskTag := restClient.DeleteRecord(fmt.Sprintf("/rest/v1/VirDomain/%s", uuid), -1, ctx)
	taskTag.WaitTask(*restClient, ctx)
}

func testAccHypercoreVMResourceRemovedConfig(vmName string) string {
	return fmt.Sprintf(`
resource "hypercore_vm" "test" {
  name              = %[1]q
  description       = "testtf acceptance test VM, safe to delete"
  tags              = ["testtf"] # with no tags, refresh reads back [""] and the plan never settles
  vcpu              = 1
  memory            = 512
  affinity_strategy = {}
}
`, vmName)
}

// A disk deleted outside Terraform must also be dropped from state (issue #112 follow-up).
func TestAccHypercoreDiskResource_RemovedOutsideTerraform(t *testing.T) {
	vmName := "testtf-vm-disk-removed-outside"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccHypercoreDiskResourceRemovedConfig(vmName),
				Check:  resource.TestCheckResourceAttr("hypercore_disk.test", "type", "VIRTIO_DISK"),
			},
			{
				PreConfig:          func() { deleteDiskOfVM(t, vmName) },
				Config:             testAccHypercoreDiskResourceRemovedConfig(vmName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func deleteDiskOfVM(t *testing.T, vmName string) {
	restClient := SetupRestClient()
	vms := utils.GetVM(map[string]any{"name": vmName}, *restClient)
	if len(vms) != 1 {
		t.Fatalf("expected exactly one VM named %s, got %d", vmName, len(vms))
	}
	disks := utils.AnyToListOfMap(vms[0]["blockDevs"])
	if len(disks) != 1 {
		t.Fatalf("expected exactly one disk on VM %s, got %d", vmName, len(disks))
	}
	ctx := context.Background()
	taskTag := restClient.DeleteRecord(fmt.Sprintf("/rest/v1/VirDomainBlockDevice/%s", utils.AnyToString(disks[0]["uuid"])), -1, ctx)
	taskTag.WaitTask(*restClient, ctx)
}

func testAccHypercoreDiskResourceRemovedConfig(vmName string) string {
	return fmt.Sprintf(`
resource "hypercore_vm" "test" {
  name              = %[1]q
  description       = "testtf acceptance test VM, safe to delete"
  tags              = ["testtf"] # with no tags, refresh reads back [""] and the plan never settles
  vcpu              = 1
  memory            = 512
  affinity_strategy = {}
}

resource "hypercore_disk" "test" {
  vm_uuid = hypercore_vm.test.id
  type    = "VIRTIO_DISK"
  size    = 1
}
`, vmName)
}
