locals {
  vm_name = "myvm"
}

data "hypercore_vm" "isovm" {
  name = local.vm_name
}

// Upload ISO from local machine
resource "hypercore_iso" "iso_upload_local" {
  name       = "testiso-local.iso"
  source_url = "file:////home/bla/Downloads/mytestiso.iso"
}

// Upload ISO from remote machine/storage
resource "hypercore_iso" "iso_upload_from_url" {
  name       = "testiso-remote.iso"
  source_url = "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/aarch64/alpine-virt-3.21.3-aarch64.iso"
}


output "uploaded_iso_LOCAL" {
  value = hypercore_iso.iso_upload_local
}

output "uploaded_iso_EXTERNAL" {
  value = hypercore_iso.iso_upload_from_url
}

// We can then use this ISO for IDE_CDROM disk
resource "hypercore_disk" "iso_attach" {
  vm_uuid  = local.vm_name
  type     = "IDE_CDROM"
  iso_uuid = hypercore_iso.iso_upload_local.id
}

output "iso_attach" {
  value = hypercore_disk.iso_attach
}
