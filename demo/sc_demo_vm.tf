# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# After we have virtual disk, we use it to create new VM from it
# First create VM without any disk.
resource "hypercore_vm" "demo_vm" {
  tags       = ["testtf"]
  name        = local.vm_name
  description = "Demo Ana's cloned VM"
  vcpu        = 4
  memory      = 4096  # MiB
  clone = {
    source_vm_uuid = data.hypercore_vms.template_vm.vms.0.uuid
    meta_data = templatefile("assets/meta-data.ubuntu-22.04.yml.tftpl", {
      name = local.vm_name,
    })
    user_data = templatefile("assets/user-data.ubuntu-22.04.yml.tftpl", {
      name                = local.vm_name,
      ssh_authorized_keys = "",
      ssh_import_id       = "",
    })
  }

  snapshot_schedule_uuid = hypercore_vm_snapshot_schedule.demo1.id
  # TODO update, "" -> null

  # Pin VM to the first node in cluster
  # If preferred_node fails, run VM on any other node.
  affinity_strategy = {
    strict_affinity = true
    preferred_node_uuid = data.hypercore_nodes.node_1.nodes.0.uuid
    backup_node_uuid = ""
    # backup_node_uuid = data.hypercore_nodes.node_2.id
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

resource "hypercore_disk" "iso" {
  vm_uuid                = hypercore_vm.demo_vm.id
  type                   = "IDE_CDROM"
  iso_uuid               = hypercore_iso.alpine_virt.id
  // TODO size, should be computed
  size     = 0.066060288
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
    hypercore_disk.iso,
    hypercore_nic.vlan_all,
    hypercore_vm_boot_order.demo_vm_boot_order,
  ]
}

resource "hypercore_vm_boot_order" "demo_vm_boot_order" {
  vm_uuid = hypercore_vm.demo_vm.id
  boot_devices = [
    hypercore_disk.os.id,
    hypercore_disk.iso.id,
    hypercore_nic.vlan_all.id,
  ]

  depends_on = [
    hypercore_disk.os,
    hypercore_disk.iso,
    hypercore_nic.vlan_all,
  ]
}
