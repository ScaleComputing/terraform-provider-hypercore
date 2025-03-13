# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "hypercore_vm" "templatevm" {
  name = "ubuntu-22.04-server-cloudimg-amd64.img"
}

output "templatevm_uuid" {
  value = data.hypercore_vm.templatevm.vms.0.uuid
}
