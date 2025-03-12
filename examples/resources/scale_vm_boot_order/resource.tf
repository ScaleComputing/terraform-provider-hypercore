locals {
  populated_vm = "example-disk-vm"
  empty_vm     = "example-empty-vm"
}

data "scale_vm" "diskvm" {
  name = local.populated_vm
}

data "scale_vm" "empty" {
  name = local.empty_vm
}

resource "scale_virtual_disk" "vd_import_os" {
  name = "example-ubuntu.img"
}

resource "scale_nic" "example_nic" {
  vm_uuid = data.scale_vm.empty.vms.0.uuid
  vlan    = 11
  type    = "VIRTIO"
}

resource "scale_disk" "os" {
  vm_uuid                = data.scale_vm.empty.vms.0.uuid
  type                   = "VIRTIO_DISK"
  size                   = 42.0
  source_virtual_disk_id = scale_virtual_disk.vd_import_os.id

  depends_on = [scale_nic.some_nic]
}

import {
  to = scale_virtual_disk.vd_import_os
  id = "16afa2e6-9ce7-4793-bb02-ede7ea32f988"
}

resource "scale_disk" "another_disk" {
  vm_uuid = data.scale_vm.empty.vms.0.uuid
  type    = "IDE_DISK"
  size    = 3.14

  depends_on = [scale_disk.os]
}

# On a VM with no disks at all. Disks were created and attached here
resource "scale_vm_boot_order" "testtf_created_boot_order" {
  vm_uuid = data.scale_vm.empty.vms.0.uuid
  boot_devices = [ # must be provided in the wanted boot order
    scale_disk.os.id,
    scale_nic.example_nic.id,
    scale_disk.another_disk.id,
  ]

  depends_on = [
    scale_disk.os,
    scale_disk.another_disk,
    scale_nic.some_nic,
  ]
}

# On a VM with already existing boot order and now modified
resource "scale_vm_boot_order" "testtf_imported_boot_order" {
  vm_uuid = data.scale_vm.diskvm.vms.0.uuid
  boot_devices = [
    "c801157d-d454-4842-88ea-d8461e9b802f",
    "ce837222-e4da-40b5-9d12-abdc5f6f73ae",
    "5c566e31-44a1-4619-9490-5403e906b2ab",
  ]
}

import {
  to = scale_vm_boot_order.testtf_imported_boot_order
  id = data.scale_vm.diskvm.vms.0.uuid
}
