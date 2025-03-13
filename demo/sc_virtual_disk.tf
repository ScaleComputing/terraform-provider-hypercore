# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "hypercore_virtual_disk" "ubuntu_2204" {
  name       = "jammy-server-cloudimg-amd64.img"
  source_url = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
}

output "vd_ubuntu_2204" {
  value = hypercore_virtual_disk.ubuntu_2204
}
