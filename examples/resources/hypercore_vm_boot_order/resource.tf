locals {
  populated_vm = "example-disk-vm"
  empty_vm     = "example-empty-vm"
}

data "hypercore_vms" "diskvm" {
  name = local.populated_vm
}

data "hypercore_vms" "empty" {
  name = local.empty_vm
}

resource "hypercore_virtual_disk" "vd_import_os" {
  name = "example-ubuntu.img"
}

resource "hypercore_nic" "example_nic" {
  vm_uuid = data.hypercore_vms.empty.vms.0.uuid
  vlan    = 11
  type    = "VIRTIO"
}

resource "hypercore_disk" "os" {
  vm_uuid                = data.hypercore_vms.empty.vms.0.uuid
  type                   = "VIRTIO_DISK"
  size                   = 42.0
  source_virtual_disk_id = hypercore_virtual_disk.vd_import_os.id

  depends_on = [hypercore_nic.some_nic]
}

import {
  to = hypercore_virtual_disk.vd_import_os
  id = "16afa2e6-9ce7-4793-bb02-ede7ea32f988"
}

resource "hypercore_disk" "another_disk" {
  vm_uuid = data.hypercore_vms.empty.vms.0.uuid
  type    = "IDE_DISK"
  size    = 3.14

  depends_on = [hypercore_disk.os]
}

# On a VM with no disks at all. Disks were created and attached here
resource "hypercore_vm_boot_order" "testtf_created_boot_order" {
  vm_uuid = data.hypercore_vms.empty.vms.0.uuid
  boot_devices = [ # must be provided in the wanted boot order
    hypercore_disk.os.id,
    hypercore_nic.example_nic.id,
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
  vm_uuid = data.hypercore_vms.diskvm.vms.0.uuid
  boot_devices = [
    "c801157d-d454-4842-88ea-d8461e9b802f",
    "ce837222-e4da-40b5-9d12-abdc5f6f73ae",
    "5c566e31-44a1-4619-9490-5403e906b2ab",
  ]
}

import {
  to = hypercore_vm_boot_order.testtf_imported_boot_order
  id = data.hypercore_vms.diskvm.vms.0.uuid
}
