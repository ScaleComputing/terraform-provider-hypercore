locals {
  vm_meta_data_tmpl = "./assets/meta-data.ubuntu-22.04.yml.tftpl"
  vm_user_data_tmpl = "./assets/user-data.ubuntu-22.04.yml.tftpl"
  vm_name           = "my-vm"
}

data "hypercore_vm" "clone_source_vm" {
  name = "source_vm"
}

resource "hypercore_vm" "myvm" {
  group       = "my-group"
  name        = local.vm_name
  description = "some description"

  vcpu   = 4
  memory = 4096 # MiB

  clone = {
    source_vm_uuid = data.hypercore_vm.clone_source_vm.vms.0.uuid
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

output "vm_uuid" {
  value = hypercore_vm.myvm.id
}
