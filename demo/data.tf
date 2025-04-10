# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

data "hypercore_vm" "template_vm" {
  name = local.template_vm_name
}

# The first node in cluster (id is 1-based index)
data "hypercore_node" "node_1" {
  peer_id = 1
}
# data "hypercore_node" "node_2" {
#   # node_2 will be a "backup node"
#   # It is same as node_1, because we have a single-node cluster.
#   peer_id = 1
# }

data "hypercore_remote_cluster_connection" "clusters_all" {
  # remote_cluster_name = "cluster-a"
}
