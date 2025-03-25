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
  vm_name         = "testtf-ana"
  another_vm_name = "testtf-ana-3"
  create_vm_name  = "testtf-ana-scheduled"
}


data "hypercore_vm" "snapvm" {
  name = local.vm_name
}

output "testtf-ana" {
  value = data.hypercore_vm.snapvm.vms.0.snapshot_schedule_uuid
}

data "hypercore_vm" "another_snapvm_schedule" {
  name = local.another_vm_name
}

resource "hypercore_vm_snapshot" "snapshot" {
  vm_uuid = data.hypercore_vm.snapvm.vms.0.uuid
  label   = "testtf-ana-snapshot"
}

resource "hypercore_vm_snapshot" "imported-snapshot" {
  vm_uuid = data.hypercore_vm.snapvm.vms.0.uuid
}

import {
  to = hypercore_vm_snapshot.imported-snapshot
  id = "24ab2255-ca77-49ec-bc96-f469cec3affb"
}

resource "hypercore_vm_snapshot_schedule" "testtf-schedule" {
  name = "testtf-schedule-2"
  rules = [
    {
      name                    = "testtf-rule-1",
      start_timestamp         = "2023-02-01 00:00:00",
      frequency               = "FREQ=MINUTELY;INTERVAL=1",
      local_retention_seconds = 300
    },
    {
      name                    = "testtf-rule-2",
      start_timestamp         = "2023-02-01 00:00:00",
      frequency               = "FREQ=MINUTELY;INTERVAL=1",
      local_retention_seconds = 300
    }
  ]
}

resource "hypercore_vm" "testtf-ana-scheduled" {
  group                  = "testtfxlab"
  name                   = local.create_vm_name
  description            = "Testing terraform resources"
  vcpu                   = 4
  memory                 = 4096 # MiB
  snapshot_schedule_uuid = hypercore_vm_snapshot_schedule.testtf-schedule.id

  clone = {
    meta_data      = ""
    source_vm_uuid = ""
    user_data      = ""
  }

  depends_on = [
    hypercore_vm_snapshot_schedule.testtf-schedule # make sure the schedule was created first
  ]
}

output "testtf-ana-scheduled" {
  value = hypercore_vm.testtf-ana-scheduled.snapshot_schedule_uuid
}

resource "hypercore_vm_snapshot_schedule" "testtf-schedule-no-rules" {
  name = "testtf-schedule-no-rules-3"
}

resource "hypercore_vm_snapshot_schedule" "testtf-schedule-imported" {
  name = "testtf-existing-schedule"
}

import {
  to = hypercore_vm_snapshot_schedule.testtf-schedule-imported
  id = "69b21f14-6bb6-4dd5-a6bc-6dec9bd59c96"
}
