locals {
  vm_name         = "example-vm"
  another_vm_name = "example-vm-two"
}

data "hypercore_vm" "example-vm" {
  name = local.vm_name
}

data "hypercore_vm" "example-vm-two" {
  name = local.another_vm_name
}

resource "hypercore_vm_replication" "example-replication" {
  vm_uuid = data.hypercore_vm.vm-repl.vms.0.uuid
  label   = "my-example-replication"

  connection_uuid = "6ab8c456-85af-4c97-8cb7-76246552b1e6" # remote connection UUID
  enable          = false
}

resource "hypercore_vm_replication" "example-replication-imported" {
  vm_uuid = data.hypercore_vm.example-vm-two.vms.0.uuid
}

import {
  to = hypercore_vm_replication.example-replication-imported
  id = "7eb23160-2c80-4519-b23d-b43fb3ca9da4"
}
