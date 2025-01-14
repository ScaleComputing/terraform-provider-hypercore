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
  vm_name           = "myvm"
  vm_network_iface  = "ens3"
  vm_network_mode   = "dhcp"
}

resource "scale_vm" "myvm" {
  group          = "vmgroup"
  name           = local.vm_name
  source_vm_name = "ubuntu-22.04-server-cloudimg-amd64.img"
  description    = "some description"

  vcpu      = 4
  memory    = 4096 # MiB
  disk_size = 20   # GB
  nics = [
    { type = "virtio" },
    { type = "INTEL_E1000", vlan = 10 }
  ]

  network_iface = local.vm_network_iface
  network_mode  = local.vm_network_mode

  power_state = "started"
  meta_data = templatefile(local.vm_meta_data_tmpl, {
    name          = local.vm_name,
    network_iface = local.vm_network_iface,
  })
  user_data = templatefile(local.vm_user_data_tmpl, {
    name                = local.vm_name,
    ssh_authorized_keys = "",
    ssh_import_id       = "",
  })
}

output "vm_list" {
  value = jsondecode(scale_vm.myvm.vm_list)
}
