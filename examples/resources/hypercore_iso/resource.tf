locals {
  vm_name = "myvm"
}

data "hypercore_vm" "isovm" {
  name = local.vm_name
}

resource "hypercore_iso" "iso_upload_local" {
  name       = "testiso-local.iso"
  source_url = "file:////home/bla/Downloads/mytestiso.iso"
}

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

