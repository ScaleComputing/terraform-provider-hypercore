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
  vm_name     = "testtf-ana-tags-2"
  src_vm_name = "ana"
}

provider "hypercore" {}

data "hypercore_vms" "src_empty" {
  name = local.src_vm_name
}

resource "hypercore_vm" "vm_on" {
  tags        = ["ana-tftag2"]
  name        = local.vm_name
  description = "VM created from scratch"
  vcpu        = 1
  memory      = 1234 # MiB

  clone = {
    source_vm_uuid = data.hypercore_vms.src_empty.vms.0.uuid
    meta_data      = ""
    user_data      = ""
  }
}
