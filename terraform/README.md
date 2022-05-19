## Terraform for multi-nic infrastructure
This terraform will 
- create security group with inbound/outbound rules to allow internal security group communication and pod subnet (`podnet`) (default: 192.168.0.0/16).
- add daemon port (default: `11000`) rule to security group in main interface
- create subnets upto the number specified in `subnet_num`.
- attach created subnets to vsi listed in `worker_names`.
#### Steps
1. Install terraform version `>= 0.1.3`
2. Modify `terraform.tfvars.template`, then copy to `.tfvars` file
    ```bash
    cp terraform.tfvars.template terraform.tfvars
    ```
3. Init terraform 
    ```bash
    terraform init
    ```
4. Initailly apply with required target (subnets)
    ```bash
    terraform apply -var-file=terraform.tfvars -target="ibm_is_subnet.subnets"
    ```
5. Apply all targets
    ```bash
    terraform apply -var-file=terraform.tfvars
    ```