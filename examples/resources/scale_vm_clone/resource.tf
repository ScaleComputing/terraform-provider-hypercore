locals {
  vm_meta_data_tmpl = "./assets/meta-data.ubuntu-22.04.yml.tftpl"
  vm_user_data_tmpl = "./assets/user-data.ubuntu-22.04.yml.tftpl"
  vm_name           = "my-ubuntu-vm"
}

resource "scale_vm_clone" "myvm" {
  group       = "vmgroup"
  name        = local.vm_name
  description = "some description"

  vcpu   = 4
  memory = 4096 # MiB

  disks = [
    {
      size = 2.5, # GB
      type = "VIRTIO_DISK",
      slot = 2,
    },
    {
      size = 2.5, # GB
      type = "VIRTIO_DISK",
      slot = 3,
    }
  ]

  nics = [
    { type = "virtio" },
    { type = "INTEL_E1000", vlan = 10 }
  ]

  power_state = "started"
  clone = {
    source_vm_uuid = "example-uuid"
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
  value = scale_vm_clone.myvm.id
}
