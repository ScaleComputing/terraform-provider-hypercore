locals {
  vm_name = "my-vm"
}

data "hypercore_vms" "myvm" {
  name = local.vm_name
}

resource "hypercore_virtual_disk" "vd_upload_local" {
  name       = "virtual-disk-local.img"
  source_url = "file:////media/testtf-local-virtual-disk.img" # 4 slashes, because /media is in the root
}

resource "hypercore_virtual_disk" "vd_upload_from_url" {
  name       = "virtual-disk-from-url.img"
  source_url = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
}

resource "hypercore_virtual_disk" "vd_import_existing" {
  name = "some-existing-virtual-disk.img"
}

# An existing virtual disk can also be imported from HC3
import {
  to = hypercore_virtual_disk.vd_import_existing

  # id has a single component - only virtual disk uuid
  id = "11424aec-0511-41c2-8be9-7fd9fb5e5138"
}

output "uploaded_vd_LOCAL" {
  value = hypercore_virtual_disk.vd_upload_local
}

output "uploaded_vd_EXTERNAL" {
  value = hypercore_virtual_disk.vd_upload_from_url
}

output "uploaded_vd_EXISTING" {
  value = hypercore_virtual_disk.vd_testtf_import_existing
}

resource "hypercore_disk" "os" {
  vm_uuid                = data.hypercore_vms.myvm.vms.0.uuid
  type                   = "VIRTIO_DISK"
  size                   = 3.4 # GB
  source_virtual_disk_id = hypercore_virtual_disk.vd_import_existing.id
}

output "created_disk" {
  value = hypercore_disk.os.id
}
