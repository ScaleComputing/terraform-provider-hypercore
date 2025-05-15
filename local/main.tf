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
  vm_name = "testtf-ana-tags"
}

provider "hypercore" {}

data "hypercore_vms" "testtf-ana-tags" {
  name = "testtf-ana-tags"
}

resource "hypercore_disk" "disk_newly_created" {
  vm_uuid        = data.hypercore_vms.testtf-ana-tags.vms.0.uuid
  type           = "IDE_DISK"
  size           = 3.0
  flash_priority = 3
}

resource "hypercore_disk" "disk_cloned" {
  vm_uuid = data.hypercore_vms.testtf-ana-tags.vms.0.uuid
  type    = "IDE_DISK"
  size    = 3.0

  depends_on = [hypercore_disk.disk_newly_created]
}

import {
  to = hypercore_disk.disk_cloned
  id = format("%s:%s:%d", data.hypercore_vms.testtf-ana-tags.vms.0.uuid, "IDE_DISK", 0)
}
