data "hypercore_remote_cluster_connections" "all_clusters" {}

data "hypercore_remote_cluster_connections" "cluster-a" {
  remote_cluster_name = "cluster-a"
}

output "all_remote_clusters" {
  value = jsonencode(data.hypercore_remote_cluster_connections.all_clusters)
}

output "filtered_remote_cluster" {
  value = jsonencode(data.hypercore_remote_cluster_connections.cluster-a)
}
