locals {
  vm_name = "myvm"
}

data "hypercore_vm" "nicvm" {
  name = local.vm_name
}

resource "hypercore_nic" "net_cloned" {
  vm_uuid = data.hypercore_vm.nicvm.vms.0.uuid
  vlan    = 10
  type    = "VIRTIO"
}

resource "hypercore_nic" "net_newly_created" {
  vm_uuid = data.hypercore_vm.nicvm.vms.0.uuid
  vlan    = 11
  type    = "VIRTIO"
}

output "nicvm_uuid" {
  value = data.hypercore_vm.nicvm.vms.0.uuid
}

# an existing NIC state can also be imported so it can then be modified
import {
  to = hypercore_nic.net_cloned

  # import id consists of three parts: vm_uuid:nic_type:nic_slot
  id = format("%s:%s:%d", data.hypercore_vm.nicvm.vms.0.uuid, "INTEL_E1000", 1)
}
