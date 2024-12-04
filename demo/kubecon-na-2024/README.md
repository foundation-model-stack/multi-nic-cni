# Multi-NIC CNI Demo

[![Dressing-up Your Cluster for AI in Minutes with a Portable Network CR - Sunyanan Choochotkaew & Tatsuhiro Chiba, IBM Research](./img/cover.png)](https://youtu.be/Sj2nBKcOWlI?si=63uQ2-RuUHQivzwm)

## System Description
- Cluster: multi-nic-cni
- Pre-installation
    - Benchmark operator (CPE)
        - Metric server enablement
        - MPI operator
            
            ```
            kubectl create -f mpi-operator.yaml
            ```

    - Grafana with thanos-querier datasource

## Required actions
- Build and replace OSU benchmark image

# Demo Steps
1. Show start state
    
    1.1. Open grafana dashboard

    1.2. Login to node
        
    ```bash
    > ip -br -c link show|grep ens
    ens3             UP             02:00:02:56:f5:c5 <BROADCAST,MULTICAST,UP,LOWER_UP> 
    ens4             UP             02:00:03:57:24:11 <BROADCAST,MULTICAST,UP,LOWER_UP> 
    ens5             UP             02:00:03:57:24:12 <BROADCAST,MULTICAST,UP,LOWER_UP>
    > ip r
    default via 10.244.0.1 dev br-ex proto dhcp src 10.244.0.4 metric 48 
    10.128.0.0/14 via 10.130.2.1 dev ovn-k8s-mp0 
    10.130.2.0/23 dev ovn-k8s-mp0 proto kernel scope link src 10.130.2.2 
    10.244.0.0/24 dev br-ex proto kernel scope link src 10.244.0.4 metric 48 
    10.244.2.0/24 dev ens4 proto kernel scope link src 10.244.2.5 metric 101 
    10.244.3.0/24 dev ens5 proto kernel scope link src 10.244.3.5 metric 102 
    169.254.169.0/29 dev br-ex proto kernel scope link src 169.254.169.2 
    169.254.169.1 dev br-ex src 10.244.0.4 
    169.254.169.3 via 10.130.2.1 dev ovn-k8s-mp0 
    172.30.0.0/16 via 169.254.169.4 dev br-ex mtu 1400
    ```

    1.3. HostInterface CR is auto-created.

    1.4. No CIDR CR

2. Deploy MultiNicNetwork

3. Show CIDR and node route

    ```bash
    > ip rule
    > ip r show table multi-nic-cni-operator-ipvlanl3
    ```

2. Deploy mpilat.yaml

    ```bash
    oc create -f mpilat.yaml
    ```

3. Waiting for job complete
    
    ```bash
    watch oc get benchmark mpilat -o=jsonpath='{.status.jobCompleted}'
    ```
    
3. Revisit grafana dashboard for result