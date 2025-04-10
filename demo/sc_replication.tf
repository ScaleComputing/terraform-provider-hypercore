# # Copyright (c) HashiCorp, Inc.
# # SPDX-License-Identifier: MPL-2.0

resource "hypercore_vm_replication" "testtf-replication" {
  vm_uuid = hypercore_vm.demo_vm.id
  label   = "testtf-demo_vm-replication"

  connection_uuid =  data.hypercore_remote_cluster_connection.clusters_all.remote_clusters.0.uuid
  enable          = true

  # If testing with replication localhost - added the connection to itself
  # - become two vm_uuid's when searching by vm by name. One is replication so vm_uuid would change
  # - when actually replicating (with two different clusters), this "ignore_changes" wouldn't be necessary
  lifecycle {
    ignore_changes = [vm_uuid]
   }
}
