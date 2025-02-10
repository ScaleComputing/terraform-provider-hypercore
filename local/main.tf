# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    scale = {
      source = "local/xlab/scale"
    }
  }
}

provider "scale" {}

locals {
  vm_meta_data_tmpl = "./assets/meta-data.ubuntu-22.04.yml.tftpl"
  vm_user_data_tmpl = "./assets/user-data.ubuntu-22.04.yml.tftpl"
  vm_name           = "testtf-myvm"
}

data "scale_vm" "templatevm" {
  name = "ubuntu-22.04-server-cloudimg-amd64.img"
}

resource "scale_vm_clone" "myvm" {
  group       = "ananas"
  name        = local.vm_name
  description = "Ana's cloned VM"

  vcpu      = 4
  memory    = 4096 # MiB
  disk_size = 20   # GB
  nics = [
    { type = "virtio" },
    { type = "INTEL_E1000", vlan = 10 }
  ]

  power_state = "stop"
  clone = {
    source_vm_uuid = data.scale_vm.templatevm.vms.0.uuid
    meta_data = templatefile(local.vm_meta_data_tmpl, {
      name = local.vm_name,
    })
    user_data = templatefile(local.vm_user_data_tmpl, {
      name                = local.vm_name,
      ssh_authorized_keys = "",
      ssh_import_id       = "",
    })
  }
}

output "vm_uuid" {
  value = scale_vm_clone.myvm.id
}
output "templatevm_uuid" {
  value = data.scale_vm.templatevm.vms.0.uuid
}
