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
  vm_name = "testtf-disk-ana"
  # vm_uuid = data.scale_vm.nicvm.vms.0.uuid
}

data "scale_vm" "diskvm" {
  # name = "testtf-nic-justin"
  name = local.vm_name
}

resource "scale_disk" "disk_cloned" {
  vm_uuid = data.scale_vm.diskvm.vms.0.uuid
  # slot    = 10
  type    = "VIRTIO_DISK"
  size    = 47.2
}

resource "scale_disk" "disk_aa" {
  vm_uuid = data.scale_vm.diskvm.vms.0.uuid
  # slot    = 11
  type    = "VIRTIO_DISK"
  size    = 3.0
}

output "diskvm_uuid" {
  value = data.scale_vm.diskvm.vms.0.uuid
}

import {
  to = scale_disk.disk_cloned
  # id = "/dev/sdh:vol-049df67901:i-12345678"
  id = format("%s:%s:%d", data.scale_vm.diskvm.vms.0.uuid, "VIRTIO_DISK", 1)
}
