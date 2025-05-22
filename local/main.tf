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
  src_vm_name = "testtf-src-empty"
  vm_name = "testtf-justin-affinity"
}

provider "hypercore" {}

data "hypercore_vms" "srcvm" {
  name = local.src_vm_name
}

resource "hypercore_vm" "myvm" {
  name = local.vm_name
  clone = {
    source_vm_uuid = data.hypercore_vms.srcvm.vms.0.uuid
    user_data = ""
    meta_data = ""
  }
  # TODO - are computed, on HC3 side
  memory = 1024
  tags = [""]
  vcpu = 1
  description = ""
  affinity_strategy = {
    # strict_affinity = true
    # preferred_node_uuid = "d676b39c-595f-4c3b-a8df-a18f308243c0"
  }
}

resource "hypercore_vm_power_state" "myvm" {
  vm_uuid = hypercore_vm.myvm.id
  # state   = "SHUTOFF" # available states are: SHUTOFF, RUNNING, PAUSED
  state   = "RUNNING" # available states are: SHUTOFF, RUNNING, PAUSED
}
