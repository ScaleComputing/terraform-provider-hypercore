# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    hypercore = {
      source = "local/xlab/hypercore"
    }
  }
}

locals {
  vm_name = "testtf-remove-running"
  src_vm_name = "testtf-src-empty"
}

provider "hypercore" {}

data "hypercore_vms" "no-such-vm" {
  name = "no-such-vm"
}

# resource "hypercore_vm" "vm_on" {
#   group       = "testtf"
#   name        = local.vm_name
#   description = "VM created from scratch"
#   vcpu        = 1
#   memory      = 1234  # MiB

#   # clone = {
#   #   source_vm_uuid = data.hypercore_vm.src_empty.vms.0.uuid
#   #   meta_data = ""
#   #   user_data = ""
#   # }
# }

# resource "hypercore_vm_power_state" "vm_on" {
#   vm_uuid = hypercore_vm.vm_on.id
#   state = "SHUTOFF"  // RUNNING SHUTOFF
# }

# output "vm_on_uuid" {
#   value = hypercore_vm.vm_on.id
# }
# output "power_state" {
#   value = hypercore_vm_power_state.vm_on.state
# }
