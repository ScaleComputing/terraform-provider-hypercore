# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    hypercore = {
      source = "local/xlab/hypercore"
    }
  }
}

provider "hypercore" {}

locals {
  vm_name = "testtf-ana"
}

data "hypercore_vm" "snapvm" {
  name = local.vm_name
}

resource "hypercore_vm_snapshot" "snapshot" {
  vm_uuid = data.hypercore_vm.snapvm.vms.0.uuid
  label   = "testtf-ana-snapshot-3"
  type    = "USER" # can be USER, AUTOMATED, SUPPORT
}

resource "hypercore_vm_snapshot" "imported-snapshot" {
  vm_uuid = data.hypercore_vm.snapvm.vms.0.uuid
}

import {
  to = hypercore_vm_snapshot.imported-snapshot
  id = "b6cc2257-d61b-4461-b3e3-2c8fab3e8614"
}

# NOTE: What a snapshot schedule will look like
# resource "hypercore_vm_snapshot" "scheduled_snapshot" {
#   vm_uuid = data.hypercore_vm.snapvm.vms.0.uuid
#   type = "AUTOMATED"  # can be USER, AUTOMATED, SUPPORT
# 
#   # usable only if type is AUTOMATED
#   # schedule_uuid = hypercore_snapshot_schedule.testtf-schedule.id
# }
# 
# resource "hypercore_vm_snapshot_schedule" "testtf-schedule" {
#   name = "schedule-name"
#   rules = [
#     {
#       start_time = "2025-01-01 13:58:16",
#       frequency = "MINUTELY",  # SECONDLY, MINUTELY, HOURLY, DAILY, WEEKLY, MONTHLY, YEARLY
#       interval = "5"
#       keep_snapshot_for_seconds = 10
#     }
#   ]
# }
