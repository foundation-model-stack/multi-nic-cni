variable "vpc_name" {}
variable "subnet_count" {}
variable "zone" {}
variable "resource_group" {}
variable "podnet" { default = "192.168.0.0/16" }
variable "daemonport" { default = 11000 }

variable "main_iface_sg_name" {}

data "ibm_is_security_group" "main_iface_sg" {
  name = var.main_iface_sg_name
}

resource "ibm_is_security_group_rule" "daemonport_inbound_rule" {
  group     = data.ibm_is_security_group.main_iface_sg.id
  direction = "inbound"
  remote    = data.ibm_is_security_group.main_iface_sg.id
  tcp {
    port_min = var.daemonport
    port_max = var.daemonport
  }
}

data "ibm_is_vpc" "vpc" {
  name = var.vpc_name
}

data "ibm_resource_group" "rg" {
  name = var.resource_group
}

resource "ibm_is_security_group" "sg" {
  resource_group = data.ibm_resource_group.rg.id
  name = "${var.vpc_name}-${var.zone}-sg-for-multi-nic-cni"
  vpc  = data.ibm_is_vpc.vpc.id
}

resource "ibm_is_security_group_rule" "intra_inbound_rule_all" {
  group     = ibm_is_security_group.sg.id
  direction = "inbound"
  remote    = ibm_is_security_group.sg.id
}

resource "ibm_is_security_group_rule" "intra_outbound_rule_all" {
  group     = ibm_is_security_group.sg.id
  direction = "outbound"
  remote    = ibm_is_security_group.sg.id
}

resource "ibm_is_security_group_rule" "podnet_inbound_rule_all" {
  group     = ibm_is_security_group.sg.id
  direction = "inbound"
  remote    = var.podnet
}

resource "ibm_is_security_group_rule" "podnet_outbound_rule_all" {
  group     = ibm_is_security_group.sg.id
  direction = "outbound"
  remote    = var.podnet
}

# create new subnets
resource "ibm_is_subnet" "subnets" {
  count = var.subnet_count
  name = "${var.vpc_name}-${var.zone}-s${count.index}"
  vpc  = data.ibm_is_vpc.vpc.id
  zone = var.zone
  total_ipv4_address_count = 256
  resource_group = data.ibm_resource_group.rg.id
}
