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
  # Use image AlmaLinux-10-GenericCloud-10.1-20251125.0.x86_64_v2.qcow2
  src_vm_name = "testtf-src-alma-10"
  vm_name = "testtf-wait-net"
}

provider "hypercore" {}

data "hypercore_vms" "srcvm" {
  name = local.src_vm_name
}

resource "hypercore_vm" "myvm" {
  name = local.vm_name
  clone = {
    source_vm_uuid = data.hypercore_vms.srcvm.vms.0.uuid
    # meta_data = templatefile("assets/meta-data.ubuntu-22.04.yml.tftpl", {
    #   name = local.vm_name,
    # })
    # user_data = templatefile("assets/user-data.ubuntu-22.04.yml.tftpl", {
    #   name                = local.vm_name,
    #   ssh_authorized_keys = "",
    #   ssh_import_id       = "justinc1",
    # })
    user_data = file("assets/alma-10/user-data")
    meta_data = file("assets/alma-10/meta-data")
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
  wait_for_guest_net_timeout = 120
}
