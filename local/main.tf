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
}

data "hypercore_vm" "snapvm" {
  name = local.vm_name
}

data "hypercore_vm" "another_snapvm" {
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

resource "hypercore_vm_snapshot_schedule" "testtf-schedule-no-rules" {
  name = "testtf-schedule-no-rules-3"
}

resource "hypercore_vm_snapshot_schedule" "testtf-schedule-imported" {
  name = "testtf-existing-schedule"

  # apply imported schedule to VMs
  # NOTE (for testing): We have 3 VMs. One of them has this scheduled applied while the
  # other 2 have no schedules. With this config, that is overwritten:
  # Initial: VM1-has-schedule, VM2-no-schedule, VM3-no-schedule
  # Overwriten: VM1-no-schedule, VM2-has-schedule, VM3-has-schedule
  vm_uuid_list = [
    data.hypercore_vm.snapvm.vms.0.uuid,
    data.hypercore_vm.another_snapvm.vms.0.uuid,
  ]
}

import {
  to = hypercore_vm_snapshot_schedule.testtf-schedule-imported
  id = "69b21f14-6bb6-4dd5-a6bc-6dec9bd59c96"
}
