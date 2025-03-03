# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    scale = {
      source = "local/xlab/scale"
    }
  }
}

provider "scale" {}

locals {
  vm_name                = "testtf-powerstate-ana"
  vm_name_without_import = "testtf-powerstate-without-import-ana"
}

data "scale_vm" "powerstatevm" {
  name = local.vm_name
}

data "scale_vm" "powerstatevm_no_import" {
  name = local.vm_name_without_import
}

resource "scale_vm_power_state" "power_state_aa" {
  vm_uuid = data.scale_vm.powerstatevm_no_import.vms.0.uuid
  state   = "RUNNING"
}

resource "scale_vm_power_state" "power_state_cloned" {
  vm_uuid = data.scale_vm.powerstatevm.vms.0.uuid
  state   = "RUNNING" // other available states: RUNNING, PAUSED - see POST VirDomain/action
}

output "powerstatevm_uuid" {
  value = data.scale_vm.powerstatevm.vms.0.uuid
}

import {
  to = scale_vm_power_state.power_state_cloned
  # id = "/dev/sdh:vol-049df67901:i-12345678"
  id = data.scale_vm.powerstatevm.vms.0.uuid
}
