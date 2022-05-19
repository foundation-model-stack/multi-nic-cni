### Tool for check connection between secondary network interfaces
This tool will check connections regarding CIDR resource list.

It will create Iperf3 server pod for all related hosts and then sequentially run Iperf3 client job for each interfaces on each pod.


Run locally:
```bash
make local-run
```

Clean up:
```bash
make clean
```

Expected output:
```bash
###########################################
## Connection Check: multinic-sample
###########################################
FROM                            TO                               CONNECTED/TOTAL IPs                            BANDWIDTHs
gpu-dallas-d5l8c-worker-2-47lzt gpu-dallas-d5l8c-worker-2-5477j  2/2             [192.168.0.2 192.168.64.2]     [ 8.80Gbits/sec 7.81Gbits/sec]
gpu-dallas-d5l8c-worker-2-47lzt gpu-dallas-d5l8c-worker-2-6dkfv  2/2             [192.168.0.195 192.168.64.195] [ 13.1Gbits/sec 7.55Gbits/sec]
gpu-dallas-d5l8c-worker-2-47lzt gpu-dallas-d5l8c-worker-2-8wh6z  2/2             [192.168.1.3 192.168.65.3]     [ 7.32Gbits/sec 7.64Gbits/sec]
gpu-dallas-d5l8c-worker-2-47lzt gpu-dallas-d5l8c-worker-3-rfrs4  0/2             [192.168.128.1 192.168.192.1]  []
gpu-dallas-d5l8c-worker-2-47lzt gpu-dallas-d5l8c-worker-2-4czvd  2/2             [192.168.0.67 192.168.64.67]   [ 7.39Gbits/sec 8.08Gbits/sec]
gpu-dallas-d5l8c-worker-2-4czvd gpu-dallas-d5l8c-worker-2-47lzt  2/2             [192.168.0.131 192.168.64.131] [ 10.9Gbits/sec 9.79Gbits/sec]
gpu-dallas-d5l8c-worker-2-4czvd gpu-dallas-d5l8c-worker-2-5477j  2/2             [192.168.0.2 192.168.64.2]     [ 5.47Gbits/sec 4.96Gbits/sec]
gpu-dallas-d5l8c-worker-2-4czvd gpu-dallas-d5l8c-worker-2-6dkfv  2/2             [192.168.0.195 192.168.64.195] [ 8.08Gbits/sec 7.72Gbits/sec]
gpu-dallas-d5l8c-worker-2-4czvd gpu-dallas-d5l8c-worker-2-8wh6z  2/2             [192.168.1.3 192.168.65.3]     [ 7.55Gbits/sec 9.93Gbits/sec]
gpu-dallas-d5l8c-worker-2-4czvd gpu-dallas-d5l8c-worker-3-rfrs4  0/2             [192.168.128.1 192.168.192.1]  []
gpu-dallas-d5l8c-worker-2-5477j gpu-dallas-d5l8c-worker-2-6dkfv  2/2             [192.168.0.195 192.168.64.195] [ 8.37Gbits/sec 8.91Gbits/sec]
gpu-dallas-d5l8c-worker-2-5477j gpu-dallas-d5l8c-worker-2-8wh6z  2/2             [192.168.1.3 192.168.65.3]     [ 10.7Gbits/sec 5.84Gbits/sec]
gpu-dallas-d5l8c-worker-2-5477j gpu-dallas-d5l8c-worker-3-rfrs4  0/2             [192.168.128.1 192.168.192.1]  []
gpu-dallas-d5l8c-worker-2-5477j gpu-dallas-d5l8c-worker-2-47lzt  2/2             [192.168.0.131 192.168.64.131] [ 5.61Gbits/sec 9.52Gbits/sec]
gpu-dallas-d5l8c-worker-2-5477j gpu-dallas-d5l8c-worker-2-4czvd  2/2             [192.168.0.67 192.168.64.67]   [ 6.56Gbits/sec 7.09Gbits/sec]
gpu-dallas-d5l8c-worker-2-6dkfv gpu-dallas-d5l8c-worker-2-47lzt  2/2             [192.168.0.131 192.168.64.131] [ 10.5Gbits/sec 8.80Gbits/sec]
gpu-dallas-d5l8c-worker-2-6dkfv gpu-dallas-d5l8c-worker-2-4czvd  2/2             [192.168.0.67 192.168.64.67]   [ 7.02Gbits/sec 9.39Gbits/sec]
gpu-dallas-d5l8c-worker-2-6dkfv gpu-dallas-d5l8c-worker-2-5477j  2/2             [192.168.0.2 192.168.64.2]     [ 7.81Gbits/sec 7.81Gbits/sec]
gpu-dallas-d5l8c-worker-2-6dkfv gpu-dallas-d5l8c-worker-2-8wh6z  2/2             [192.168.1.3 192.168.65.3]     [ 9.79Gbits/sec 8.18Gbits/sec]
gpu-dallas-d5l8c-worker-2-6dkfv gpu-dallas-d5l8c-worker-3-rfrs4  0/2             [192.168.128.1 192.168.192.1]  []
gpu-dallas-d5l8c-worker-2-8wh6z gpu-dallas-d5l8c-worker-2-47lzt  2/2             [192.168.0.131 192.168.64.131] [ 9.52Gbits/sec 9.03Gbits/sec]
gpu-dallas-d5l8c-worker-2-8wh6z gpu-dallas-d5l8c-worker-2-4czvd  2/2             [192.168.0.67 192.168.64.67]   [ 9.65Gbits/sec 4.88Gbits/sec]
gpu-dallas-d5l8c-worker-2-8wh6z gpu-dallas-d5l8c-worker-2-5477j  2/2             [192.168.0.2 192.168.64.2]     [ 7.99Gbits/sec 7.39Gbits/sec]
gpu-dallas-d5l8c-worker-2-8wh6z gpu-dallas-d5l8c-worker-2-6dkfv  2/2             [192.168.0.195 192.168.64.195] [ 6.56Gbits/sec 6.88Gbits/sec]
gpu-dallas-d5l8c-worker-2-8wh6z gpu-dallas-d5l8c-worker-3-rfrs4  0/2             [192.168.128.1 192.168.192.1]  []
gpu-dallas-d5l8c-worker-3-rfrs4 gpu-dallas-d5l8c-worker-2-5477j  0/2             [192.168.0.2 192.168.64.2]     []
gpu-dallas-d5l8c-worker-3-rfrs4 gpu-dallas-d5l8c-worker-2-6dkfv  0/2             [192.168.0.195 192.168.64.195] []
gpu-dallas-d5l8c-worker-3-rfrs4 gpu-dallas-d5l8c-worker-2-8wh6z  0/2             [192.168.1.3 192.168.65.3]     []
gpu-dallas-d5l8c-worker-3-rfrs4 gpu-dallas-d5l8c-worker-2-47lzt  0/2             [192.168.0.131 192.168.64.131] []
gpu-dallas-d5l8c-worker-3-rfrs4 gpu-dallas-d5l8c-worker-2-4czvd  0/2             [192.168.0.67 192.168.64.67]   []
###########################################
```