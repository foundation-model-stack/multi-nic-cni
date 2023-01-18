# End-to-end Test

This test leverages [kwok](https://github.com/kubernetes-sigs/kwok) to simulate a big cluster with more than 100 nodes in minutes.


## Requirements
- make, yq, jq
- docker
- kind/kubectl

## Prepare cluster
```bash
# create kind cluster with 1000 max pods
CLUSTER_NAME=kind_1000
make create-kind
# build and load all requried images to kind cluster
make build-load-images
# deploy required controllers and CR (kwok, multi-nic-cni, net-attach-def CR)
make prepare-controller
# enable execution permission of script.sh
chmod +x script.sh
```

## Test cases

There are three test cases.
1. Scale cluster in steps from 10, 20, 50, 100 and to 200. Then, scale down with the same steps.
    ```bash
    ./script.sh test_step_scale
    ```
2. Allocate IPs for 5 pods to 10 nodes and then deallocate.
    ```bash
    ./script.sh test_allocate
    ```
3. Taint a node, allocate IPs for other available nodes, then untaint the node. Next, taint the node that already has pods deployed and then untaint it. 
    ```bash
    ./script.sh test_taint
    ```

For each state change, check corresponding MultiNicNetowrk, CIDR, and IPPool CRs.