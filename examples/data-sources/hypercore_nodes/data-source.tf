# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Get all nodes
data "hypercore_nodes" "all_nodes" {
}

output "hypercore_nodes" {
  value = data.hypercore_nodes.all_nodes.nodes
}

# Get a specific node
data "hypercore_nodes" "node_1" {
  peer_id = 1
}

output "hypercore_nodes_1_uuid" {
  value = data.hypercore_nodes.node_1.nodes.0.uuid
}
