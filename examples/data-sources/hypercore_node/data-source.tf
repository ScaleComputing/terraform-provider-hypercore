# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Get all nodes
data "hypercore_node" "all_nodes" {
}

output "hypercore_nodes" {
  value = data.hypercore_node.all_nodes.nodes
}

# Get a specific node
data "hypercore_node" "node_1" {
  peer_id = 1
}

output "hypercore_node_1_uuid" {
  value = data.hypercore_node.node_1.nodes.0.uuid
}
