# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    hypercore = {
      source = "local/xlab/hypercore"
    }
  }
}

provider "hypercore" {}

locals {
  vm_name        = "testtf-disk-ana"
  empty_vm       = "testtf-ana"
  clone_empty_vm = "testtf-clone-ana"

  vm_meta_data_tmpl = "./assets/meta-data.ubuntu-22.04.yml.tftpl"
  vm_user_data_tmpl = "./assets/user-data.ubuntu-22.04.yml.tftpl"
}

data "hypercore_vm" "diskvm" {
  name = local.vm_name
}

data "hypercore_vm" "empty" {
  name = local.empty_vm
}

output "empty_vm" {
  value = data.hypercore_vm.empty.vms.0.uuid
}

output "disk_vm" {
  value = data.hypercore_vm.diskvm.vms.0.uuid
}

resource "hypercore_vm" "clone_empty" {
  group       = "ananas-clone"
  name        = local.clone_empty_vm
  description = "some description"

  vcpu   = 4
  memory = 4096 # MiB

  clone = {
    source_vm_uuid = data.hypercore_vm.empty.vms.0.uuid
    meta_data = templatefile(local.vm_meta_data_tmpl, {
      name = local.clone_empty_vm,
    })
    user_data = templatefile(local.vm_user_data_tmpl, {
      name                = local.clone_empty_vm,
      ssh_authorized_keys = "",
      ssh_import_id       = "",
    })
  }
}

resource "hypercore_vm_power_state" "start_clone_empy" {
  vm_uuid = hypercore_vm.clone_empty.id
  state   = "RUNNING"

  depends_on = [hypercore_vm.clone_empty]
}

resource "hypercore_vm_power_state" "stop_clone_empy" {
  vm_uuid = hypercore_vm.clone_empty.id
  state   = "SHUTOFF"
  force_shutoff = true

  depends_on = [hypercore_vm_power_state.start_clone_empy]
}

resource "hypercore_virtual_disk" "vd_import_os" {
  name = "testtf-ana-ubuntu.img"
}

resource "hypercore_nic" "some_nic" {
  vm_uuid = data.hypercore_vm.empty.vms.0.uuid
  vlan    = 11
  type    = "VIRTIO"

  depends_on = [ hypercore_vm.clone_empty ]
}

resource "hypercore_disk" "os" {
  vm_uuid                = data.hypercore_vm.empty.vms.0.uuid
  type                   = "VIRTIO_DISK"
  size                   = 42.2
  source_virtual_disk_id = hypercore_virtual_disk.vd_import_os.id

  depends_on = [hypercore_nic.some_nic]
}

import {
  to = hypercore_virtual_disk.vd_import_os
  id = "4cf1cbc7-588c-4897-b0b1-d212d61e4bc5"
}

resource "hypercore_disk" "another_disk" {
  vm_uuid = data.hypercore_vm.empty.vms.0.uuid
  type    = "IDE_DISK"
  size    = 3.14

  depends_on = [hypercore_disk.os]
}

# On a VM with no disks at all. Disks were created and attached here
resource "hypercore_vm_boot_order" "testtf_created_boot_order" {
  vm_uuid = data.hypercore_vm.empty.vms.0.uuid
  boot_devices = [
    hypercore_disk.os.id,
    hypercore_nic.some_nic.id,
    hypercore_disk.another_disk.id,
  ]

  depends_on = [
    hypercore_disk.os,
    hypercore_disk.another_disk,
    hypercore_nic.some_nic,
  ]
}

# On a VM with already existing boot order and now modified
resource "hypercore_vm_boot_order" "testtf_imported_boot_order" {
  vm_uuid = data.hypercore_vm.diskvm.vms.0.uuid
  boot_devices = [
    "c801157d-d454-4842-88ea-d8461e9b802f",
    "ce837222-e4da-40b5-9d12-abdc5f6f73ae",
    "5c566e31-44a1-4619-9490-5403e906b2ab",
  ]
}

import {
  to = hypercore_vm_boot_order.testtf_imported_boot_order
  id = data.hypercore_vm.diskvm.vms.0.uuid
}
