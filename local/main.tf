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

# resource "scale_virtual_disk" "vd_upload_invalid_source" {
#   name       = "testtf-ana-virtual-disk-local.img"
#   source_url = "./assets/testtf-local-virtual-disk.img"
# }

resource "scale_virtual_disk" "vd_upload_local" {
  name       = "testtf-ana-virtual-disk-local.img"
  source_url = "file:////home/anazobec/GitHub/repos/terraform-provider-scale/local/assets/testtf-local-virtual-disk.img"
}

resource "scale_virtual_disk" "vd_upload_from_url" {
  name       = "testtf-ana-virtual-disk-from-url.img"
  source_url = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
}

resource "scale_virtual_disk" "vd_testtf_import_existing" {
  name = "testtf-ana-virtual-disk-from-url.img"
}

import {
  to = scale_virtual_disk.vd_testtf_import_existing
  # id has single component - only virtual disk name
  id = "11424aec-0511-41c2-8be9-7fd9fb5e5138"
}

output "uploaded_vd_LOCAL" {
  value = scale_virtual_disk.vd_upload_local
}

output "uploaded_vd_EXTERNAL" {
  value = scale_virtual_disk.vd_upload_from_url
}

output "uploaded_vd_EXISTING" {
  value = scale_virtual_disk.vd_testtf_import_existing
}

# # -----------------------------------------
# # After we have virtual disk, we use it to create new VM from it
# # First create VM without any disk.
# resource "scale_vm_clone" "myvm" {
#   group       = "ananas"
#   name        = local.vm_name
#   description = "Ana's cloned VM"
#   vcpu        = 4
#   memory      = 4096 # MiB
# }

# Next clone existing virtual_disk, and attach it to the VM.
# POST rest/v1/VirtualDisk/{uuid}/attach
resource "scale_disk" "os" {
  vm_uuid                = data.scale_vm.myvm.vms.0.uuid
  type                   = "VIRTIO_DISK"
  size                   = 3.4 # GB
  source_virtual_disk_id = scale_virtual_disk.vd_testtf_import_existing.id
}

output "created_disk" {
  value = scale_disk.os.id
}
