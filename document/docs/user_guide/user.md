# MultiNicNetwork Usage and Testing

## Steps
### 1. check available network

```bash
> oc get multinicnetwork

NAME                              AGE
multi-nic-cni-operator-ipvlanl3   12s
```

### 2. annotate the pod

```yaml
metadata:
  annotations:
    k8s.v1.cni.cncf.io/networks: multi-nic-sample
```

## Check connections
### One-time peer-to-peer 
This is a quick test by a randomly picked-up pair.

1. Set target peer
   
        export SERVER_HOST_NAME=<target-server-node-name>
        export CLIENT_HOST_NAME=<target-client-node-name>
    
    If the target peer is not set, the last two of listed nodes will be selected as server and client, respectively.

2. Run the test
    
        make sample-concheck

    Example output:
    
        # pod/multi-nic-iperf3-server created
        # pod/multi-nic-iperf3-server condition met
        # pod/multi-nic-iperf3-client created
        # pod/multi-nic-iperf3-client condition met
        # [  5] local 192.168.0.66 port 46284 connected to 192.168.0.130 port 5201
        # [ ID] Interval           Transfer     Bitrate         Retr  Cwnd
        # [  5]   0.00-1.00   sec   121 MBytes  1.02 Gbits/sec    0   3.04 MBytes
        # [  5]   1.00-2.00   sec   114 MBytes   954 Mbits/sec    0   3.04 MBytes
        # [  5]   2.00-3.00   sec   115 MBytes   964 Mbits/sec   45   2.19 MBytes
        # [  5]   3.00-4.00   sec   114 MBytes   954 Mbits/sec   45   1.67 MBytes
        # [  5]   4.00-5.00   sec   114 MBytes   954 Mbits/sec    0   1.77 MBytes
        # - - - - - - - - - - - - - - - - - - - - - - - - -
        # [ ID] Interval           Transfer     Bitrate         Retr
        # [  5]   0.00-5.00   sec   577 MBytes   969 Mbits/sec   90             sender
        # [  5]   0.00-5.04   sec   574 MBytes   956 Mbits/sec                  receiver

        # iperf Done.
        # pod "multi-nic-iperf3-client" deleted
        # pod "multi-nic-iperf3-server" deleted


### One-time all-to-all  
This is recommended for small cluster ( <10 HostInterfaces )

1. Run 

        make concheck

    Example output:

        # serviceaccount/multi-nic-concheck-account created
        # clusterrole.rbac.authorization.k8s.io/multi-nic-concheck-cr created
        # clusterrolebinding.rbac.authorization.k8s.io/multi-nic-concheck-cr-binding created
        # job.batch/multi-nic-concheck created
        # Wait for job/multi-nic-concheck to complete
        # job.batch/multi-nic-concheck condition met
        # 2023/02/14 01:22:21 Config
        # W0214 01:22:21.976565       1 client_config.go:617] Neither --kubeconfig nor --master was specified.  Using the inClusterConfig.  This might not work.
        # 2023/02/14 01:22:23 2/2 servers successfully created
        # 2023/02/14 01:22:23 p-cni-operator-ipvlanl3-multi-nic-n7zf6-worker-2-zt5l5-serv: Pending
        ...
        # 2023/02/14 01:23:13 2/2 clients successfully finished
        # ###########################################
        # ## Connection Check: multi-nic-cni-operator-ipvlanl3
        # ###########################################
        # FROM                           TO                              CONNECTED/TOTAL IPs                            BANDWIDTHs
        # multi-nic-n7zf6-worker-2-zt5l5 multi-nic-n7zf6-worker-2-zxw2n  2/2             [192.168.0.65 192.168.64.65]   [ 1.05Gbits/sec 1.03Gbits/sec]
        # multi-nic-n7zf6-worker-2-zxw2n multi-nic-n7zf6-worker-2-zt5l5  2/2             [192.168.0.129 192.168.64.129] [ 934Mbits/sec 937Mbits/sec]
        # ###########################################
        # 2023/02/14 01:23:13 multi-nic-cni-operator-ipvlanl3 checked

    If the job takes longer than 50 minutes (mostly in large cluster), you will get 

    `error: timed out waiting for the condition on jobs/multi-nic-concheck`
    
      Check the test progress directly:
    
        kubectl get po -w
  
      When the multi-nic-concheck job has completed, check the log:
  
        kubectl logs job/multi-nic-concheck
  
    If some connection failed, you may investigate the failure from log of the iperf3 client pod.

2. Clean up the job
   
        make clean-concheck
