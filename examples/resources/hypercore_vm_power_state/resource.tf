locals {
  vm_name                = "my-vm-to-import"
  vm_name_without_import = "my-vm-with-no-import"
}

data "hypercore_vms" "powerstatevm" {
  name = local.vm_name
}

data "hypercore_vms" "powerstatevm_no_import" {
  name = local.vm_name_without_import
}

resource "hypercore_vm_power_state" "power_state_with_no_import" {
  vm_uuid = data.hypercore_vms.powerstatevm_no_import.vms.0.uuid
  state   = "RUNNING"
}

resource "hypercore_vm_power_state" "power_state_with_import" {
  vm_uuid = data.hypercore_vms.powerstatevm.vms.0.uuid
  state   = "RUNNING" # available states are: SHUTOFF, RUNNING, PAUSED
}

output "powerstatevm_uuid" {
  value = data.hypercore_vms.powerstatevm.vms.0.uuid
}

# an existing VM's power state can be imported so it can then be modified
import {
  to = hypercore_vm_power_state.power_state_with_import

  # import id is simply the VM's UUID
  id = data.hypercore_vms.powerstatevm.vms.0.uuid
}
