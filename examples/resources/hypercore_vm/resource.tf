locals {
  vm_meta_data_tmpl = "./assets/meta-data.ubuntu-22.04.yml.tftpl"
  vm_user_data_tmpl = "./assets/user-data.ubuntu-22.04.yml.tftpl"
  vm_name           = "my-vm"
}

data "hypercore_vms" "clone_source_vm" {
  name = "source_vm"
}

resource "hypercore_vm" "myvm" {
  group       = "my-group"
  name        = local.vm_name
  description = "some description"

  vcpu                   = 4
  memory                 = 4096 # MiB
  snapshot_schedule_uuid = data.hypercore_vms.clone_source_vm.vms.0.snapshot_schedule_uuid

  clone = {
    source_vm_uuid = data.hypercore_vms.clone_source_vm.vms.0.uuid
    meta_data = templatefile(local.vm_meta_data_tmpl, {
      name = local.vm_name,
    })
    user_data = templatefile(local.vm_user_data_tmpl, {
      name                = local.vm_name,
      ssh_authorized_keys = "",
      ssh_import_id       = "",
    })
  }
}

resource "hypercore_vm" "import-from-smb" {
  group       = "my-group"
  name        = "imported-vm"
  description = "some description"

  vcpu   = 4
  memory = 4096 # MiB

  import = {
    server    = "10.5.11.39"
    username  = ";administrator"
    password  = "***"
    path      = "/cidata"
    file_name = "example-template.xml"
  }
}

output "vm_uuid" {
  value = hypercore_vm.myvm.id
}
