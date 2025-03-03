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
  vm_name           = "testtf-myvm-ana-scale-vm-rename"
}

data "scale_vm" "clone_source_vm" {
  name = "ubuntu-22.04-server-cloudimg-amd64.img"
}

resource "scale_vm" "myvm" {
  group       = "ananas"
  name        = local.vm_name
  description = "some description"

  vcpu   = 4
  memory = 4096 # MiB

  clone = {
    source_vm_uuid = data.scale_vm.clone_source_vm.vms.0.uuid
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
  value = scale_vm.myvm.id
}
