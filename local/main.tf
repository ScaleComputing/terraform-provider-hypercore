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
  vm_name        = "testtf-disk-justin"
  empty_vm       = "testtf-ana"
  clone_empty_vm = "testtf-clone-ana"

  vm_meta_data_tmpl = "./assets/meta-data.ubuntu-22.04.yml.tftpl"
  vm_user_data_tmpl = "./assets/user-data.ubuntu-22.04.yml.tftpl"
}

resource "hypercore_vm" "myvm" {
  name = local.vm_name
  clone = {
    source_vm_uuid = ""
    meta_data      = ""
    user_data      = ""
  }
  affinity_strategy = {
    strict_affinity     = true
    preferred_node_uuid = data.hypercore_node.cluster0_peer1.nodes.0.uuid
    backup_node_uuid    = data.hypercore_node.cluster0_peer1.nodes.0.uuid
  }
}

data "hypercore_node" "cluster0_all" {
}

data "hypercore_node" "cluster0_peer1" {
  peer_id = 1
}

output "myvm" {
  value = hypercore_vm.myvm
}

output "cluster_0_peer_1_uuid" {
  value = data.hypercore_node.cluster0_peer1.nodes.0.uuid
}
