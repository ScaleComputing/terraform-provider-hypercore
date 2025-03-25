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
  vm_name        = "testtf-disk-ana"
  empty_vm       = "testtf-ana"
  clone_empty_vm = "testtf-clone-ana"

  vm_meta_data_tmpl = "./assets/meta-data.ubuntu-22.04.yml.tftpl"
  vm_user_data_tmpl = "./assets/user-data.ubuntu-22.04.yml.tftpl"
}

data "hypercore_node" "cluster0_all" {
}

data "hypercore_node" "cluster0_peer1" {
  peer_id = 1
}

output "cluster_0_peer_1_uuid" {
  value = data.hypercore_node.cluster0_peer1.nodes.0.uuid
}
