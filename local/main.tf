# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    hypercore = {
      source = "local/xlab/hypercore"
    }
  }
}

locals {
  vm_name = "testtf-justin-nic-mac"
}

provider "hypercore" {}

data "hypercore_vms" "myvm" {
  name = local.vm_name
}

resource "hypercore_nic" "nic_newly_created" {
  vm_uuid        = data.hypercore_vms.myvm.vms.0.uuid
  type           = "INTEL_E1000"
  vlan           = 11
}

resource "hypercore_nic" "nic_imported" {
  vm_uuid = data.hypercore_vms.myvm.vms.0.uuid
  type    = "VIRTIO"
  vlan    = 0

  depends_on = [hypercore_nic.nic_newly_created]
}

import {
  to = hypercore_nic.nic_imported
  id = format("%s:%s:%d", data.hypercore_vms.myvm.vms.0.uuid, "VIRTIO", 0)
}
