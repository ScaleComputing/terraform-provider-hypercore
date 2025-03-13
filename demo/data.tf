# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "hypercore_vm" "template_vm" {
  name = local.template_vm_name
}
