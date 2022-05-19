output "subnets" {
  value = ibm_is_subnet.subnets
}

output "worker_subnets" {
  value = local.worker_subnets
  sensitive = true
}