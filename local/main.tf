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

data "hypercore_remote_cluster_connection" "all_clusters" {}

data "hypercore_remote_cluster_connection" "cluster-a" {
  remote_cluster_name = "cluster-a"
}

output "all_remote_clusters" {
  value = jsonencode(data.hypercore_remote_cluster_connection.all_clusters)
}

output "filtered_remote_cluster" {
  value = jsonencode(data.hypercore_remote_cluster_connection.cluster-a)
}
