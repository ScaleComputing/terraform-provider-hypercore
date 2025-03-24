# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "hypercore_iso" "alpine_virt" {
  name       = "alpine-virt-3.21.3-x86_64.iso"
  source_url = "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/x86_64/alpine-virt-3.21.3-x86_64.iso"
}

output "alpine_iso" {
  value = hypercore_iso.alpine_virt
}
