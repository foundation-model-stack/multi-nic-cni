variable "region" {}
variable "apikey" {}

provider "ibm" {
ibmcloud_api_key    = var.apikey
region = var.region
}
