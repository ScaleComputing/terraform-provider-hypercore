# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "hypercore_vm_snapshot" "snap1" {
  vm_uuid = hypercore_vm.demo_vm.id
  label   = "testtf-demo-snap1"
}

resource "hypercore_vm_snapshot_schedule" "demo1" {
  name = "testtf-demo-schedule-1"
  rules = [
    {
      name                    = "testtf-rule-1",
      start_timestamp         = "2023-01-01 01:00:00",
      frequency               = "FREQ=MINUTELY;INTERVAL=1",
      local_retention_seconds = 300
    },
    {
      name                    = "testtf-rule-2",
      start_timestamp         = "2023-02-02 02:00:00",
      frequency               = "FREQ=MINUTELY;INTERVAL=2",
      local_retention_seconds = 300
    }
  ]
}

