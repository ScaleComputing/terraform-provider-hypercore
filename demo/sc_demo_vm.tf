# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# After we have virtual disk, we use it to create new VM from it
# First create VM without any disk.
resource "hypercore_vm" "demo_vm" {
  group       = "testtf"
  name        = local.vm_name
  description = "Demo Ana's cloned VM"
  vcpu        = 4
  memory      = 4096  # MiB
  clone = {
    source_vm_uuid = data.hypercore_vm.template_vm.vms.0.uuid
    meta_data = templatefile("assets/meta-data.ubuntu-22.04.yml.tftpl", {
      name = local.vm_name,
    })
    user_data = templatefile("assets/user-data.ubuntu-22.04.yml.tftpl", {
      name                = local.vm_name,
      ssh_authorized_keys = "",
      ssh_import_id       = "",
    })
  }
}

# Next clone existing virtual_disk, and attach it to the VM.
# POST rest/v1/VirtualDisk/{uuid}/attach
resource "hypercore_disk" "os" {
  vm_uuid                = hypercore_vm.demo_vm.id
  type                   = "VIRTIO_DISK"
  size                   = 20.5  # GB
  source_virtual_disk_id = hypercore_virtual_disk.ubuntu_2204.id
}

resource "hypercore_nic" "vlan_all" {
  vm_uuid                = hypercore_vm.demo_vm.id
  type                   = "VIRTIO"
  vlan                   = 0
}

# import {
#   to = hypercore_vm_power_state.demo_vm
#   id = hypercore_vm.demo_vm.id
# }

resource "hypercore_vm_power_state" "demo_vm" {
  vm_uuid = hypercore_vm.demo_vm.id
  state   = "RUNNING" # available states are: SHUTOFF, RUNNING, PAUSED
  depends_on = [
    hypercore_disk.os,
    hypercore_nic.vlan_all,
    hypercore_vm_boot_order.demo_vm_boot_order,
  ]
}

resource "hypercore_vm_boot_order" "demo_vm_boot_order" {
  vm_uuid = hypercore_vm.demo_vm.id
  boot_devices = [
    hypercore_disk.os.id,
    hypercore_nic.vlan_all.id,
  ]

  depends_on = [
    hypercore_disk.os,
    hypercore_nic.vlan_all,
  ]
}
