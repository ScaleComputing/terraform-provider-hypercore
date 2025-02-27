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
  vm_name           = "testtf-nic-justin"
  # vm_uuid = data.scale_vm.nicvm.vms.0.uuid
}

data "scale_vm" "nicvm" {
  # name = "testtf-nic-justin"
  name = local.vm_name
}

resource "scale_nic" "net_cloned" {
  vm_uuid = data.scale_vm.nicvm.vms.0.uuid
  vlan = 10
  type = "VIRTIO"
  # macAddress = ""
}

resource "scale_nic" "net_aa" {
  vm_uuid = data.scale_vm.nicvm.vms.0.uuid
  vlan = 11
  type = "VIRTIO"
  # macAddress = ""
}

output "nicvm_uuid" {
  value = data.scale_vm.nicvm.vms.0.uuid
}

# to samo testiram, kako bi opisal imported disk
import {
    to = scale_nic.net_cloned
    # id = "/dev/sdh:vol-049df67901:i-12345678"
    id = format("%s:%s:%d", data.scale_vm.nicvm.vms.0.uuid, "INTEL_E1000", 1)
}
