locals {
  vm_name = "myvm"
}

data "hypercore_vm" "diskvm" {
  name = local.vm_name
}

resource "hypercore_disk" "disk_cloned" {
  vm_uuid = data.hypercore_vm.diskvm.vms.0.uuid
  type    = "VIRTIO_DISK"
  size    = 47.2
}

resource "hypercore_disk" "disk_newly_created" {
  vm_uuid = data.hypercore_vm.diskvm.vms.0.uuid
  type    = "IDE_DISK"
  size    = 3.0
}

output "diskvm_uuid" {
  value = data.hypercore_vm.diskvm.vms.0.uuid
}

# an existing disk state can also be import to then modify
import {
  to = hypercore_disk.disk_cloned

  # import id consists of three parts: vm_uuid:disk_type:disk_slot
  id = format("%s:%s:%d", data.hypercore_vm.diskvm.vms.0.uuid, "VIRTIO_DISK", 1)
}
