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
  vm_meta_data_tmpl = "./assets/meta-data.ubuntu-22.04.yml.tftpl"
  vm_user_data_tmpl = "./assets/user-data.ubuntu-22.04.yml.tftpl"
  vm_name           = "my-vm"
}

data "hypercore_vm" "clone_source_vm" {
  name = "source_vm"
}

# This is what the updated resource will look like with import
resource "hypercore_vm" "import-from-http" {
  group       = "my-group"
  name        = local.vm_name
  description = "some description"

  vcpu   = 4
  memory = 4096 # MiB

  import = {
    http_path = "http://someurl/my-vm"
  }
}

resource "hypercore_vm" "import-from-smb" {
  group       = "my-group"
  name        = local.vm_name
  description = "some description"

  vcpu   = 4
  memory = 4096 # MiB

  import = {
    server    = "server"
    username  = "username"
    password  = "password"
    path      = "path"
    file_name = "file_name"
  }
}

# NOTE: 'clone' parameter can still be defined along with 'import', but in THIS case, 'import' will be the one taking effect, not clone

output "vm_uuid" {
  value = hypercore_vm.myvm.id
}
