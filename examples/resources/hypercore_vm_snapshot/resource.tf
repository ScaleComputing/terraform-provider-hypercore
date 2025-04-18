locals {
  vm_name         = "example-vm-one"
  another_vm_name = "example-vm-two"
}

data "hypercore_vms" "example-vm-one" {
  name = local.vm_name
}

data "hypercore_vms" "example-vm-two" {
  name = local.another_vm_name
}

resource "hypercore_vm_snapshot" "snapshot" {
  vm_uuid = data.hypercore_vms.example-vm-one.vms.0.uuid
  label   = "my-snapshot"
}

resource "hypercore_vm_snapshot" "imported-snapshot" {
  vm_uuid = data.hypercore_vms.example-vm-two.vms.0.uuid
}

import {
  to = hypercore_vm_snapshot.imported-snapshot
  id = "24ab2255-ca77-49ec-bc96-f469cec3affb"
}
