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
  vm_name         = "testtf-ana-replication"
  another_vm_name = "testtf-ana"
}

data "hypercore_vm" "vm-repl" {
  name = local.vm_name
}

data "hypercore_vm" "vm-repl-2" {
  name = local.another_vm_name
}

resource "hypercore_vm_replication" "testtf-replication" {
  vm_uuid = data.hypercore_vm.vm-repl.vms.0.uuid
  label   = "testtf-ana-create-replication"

  connection_uuid = "6ab8c456-85af-4c97-8cb7-76246552b1e6" # remote connection UUID
  enable          = false                                  # should this default to true like in the HC3 swagger docs or make it required either way (whether it's true or false)?

  # I'm testing with replication localhost - added the connection to itself
  # - become two vm_uuid's when searching by vm by name. One is replication so vm_uuid would change
  # - when actually replicating (with two different clusters), this "ignore_changes" wouldn't be necessary
  lifecycle {
    ignore_changes = [vm_uuid]
  }
}

resource "hypercore_vm_replication" "testtf-replication-imported" {
  vm_uuid = data.hypercore_vm.vm-repl-2.vms.0.uuid

  # enable = true

  # I'm testing with replication localhost - added the connection to itself
  # - become two vm_uuid's when searching by vm by name. One is replication so vm_uuid would change
  # - when actually replicating (with two different clusters), this "ignore_changes" wouldn't be necessary
  lifecycle {
    ignore_changes = [vm_uuid]
  }
}

import {
  to = hypercore_vm_replication.testtf-replication-imported
  id = "7eb23160-2c80-4519-b23d-b43fb3ca9da4"
}
