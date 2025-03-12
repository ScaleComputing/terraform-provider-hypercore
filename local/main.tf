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
}

data "scale_vm" "myvm" {
  name = local.vm_name
}

resource "scale_virtual_disk" "vd_testtf_import_existing" {
  name = "testtf-ana-virtual-disk-from-url.img"
}

import {
  to = scale_virtual_disk.vd_testtf_import_existing
  # id has single component - only virtual disk name
  id = "11424aec-0511-41c2-8be9-7fd9fb5e5138"
}

output "uploaded_vd_EXISTING" {
  value = scale_virtual_disk.vd_testtf_import_existing
}

resource "scale_disk" "os" {
  vm_uuid                = data.scale_vm.myvm.vms.0.uuid
  type                   = "VIRTIO_DISK"
  size                   = 3.7 # GB
  source_virtual_disk_id = scale_virtual_disk.vd_testtf_import_existing.id
}

output "created_disk" {
  value = scale_disk.os.id
}
