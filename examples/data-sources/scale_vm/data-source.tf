# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "scale_vm" "templatevm" {
  name = "ubuntu-22.04-server-cloudimg-amd64.img"
}

output "templatevm_uuid" {
  value = data.scale_vm.templatevm.vms.0.uuid
}
