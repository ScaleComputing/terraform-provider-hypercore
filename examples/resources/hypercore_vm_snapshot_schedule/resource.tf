locals {
  vm_name         = "example-vm-one"
  another_vm_name = "example-vm-two"
}

data "hypercore_vm" "example-vm-one" {
  name = local.vm_name
}

data "hypercore_vm" "example-vm-two" {
  name = local.another_vm_name
}

resource "hypercore_vm_snapshot_schedule" "example-schedule" {
  name = "my-schedule"
  rules = [
    {
      name                    = "first-example-rule",
      start_timestamp         = "2023-02-01 00:00:00",
      frequency               = "FREQ=MINUTELY;INTERVAL=1",
      local_retention_seconds = 300
    },
    {
      name                    = "second-example-rule",
      start_timestamp         = "2023-02-01 00:00:00",
      frequency               = "FREQ=MINUTELY;INTERVAL=1",
      local_retention_seconds = 300
    }
  ]
}

resource "hypercore_vm_snapshot_schedule" "example-schedule-no-rules" {
  name = "my-schedule-without-rules"
}

resource "hypercore_vm_snapshot_schedule" "example-schedule-imported" {
  name = "my-imported-schedule"
}

import {
  to = hypercore_vm_snapshot_schedule.example-schedule-imported
  id = "69b21f14-6bb6-4dd5-a6bc-6dec9bd59c96"
}
