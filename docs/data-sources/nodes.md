---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "hypercore_nodes Data Source - hypercore"
subcategory: ""
description: |-
  
---

# hypercore_nodes (Data Source)



## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `peer_id` (Number)

### Read-Only

- `nodes` (Attributes List) (see [below for nested schema](#nestedatt--nodes))

<a id="nestedatt--nodes"></a>
### Nested Schema for `nodes`

Read-Only:

- `backplane_ip` (String)
- `lan_ip` (String)
- `peer_id` (Number)
- `uuid` (String)
