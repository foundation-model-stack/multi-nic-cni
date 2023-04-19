# Example Client of Multi-NIC CNI Health Checker

`read_status.py` is an example client application to read and parse response from /status API of health checker

To test locally, follow the steps below.

1. forward checker port

    ```bash
    checker=$(kubectl get po -n openshift-operators|grep multi-nic-cni-health-checker|awk '{ print $1 }')
    kubectl port-forward ${checker} -n openshift-operators 8080:8080
    ```

2. set local /status path, `CHECKER_URL`

    ```bash
    export CHECKER_URL=http://localhost:8080/status
    ```

3. run `python read_status.py`

    3.1. Get status of all hosts

    ```bash
    > python read_status.py
    Found the following potential CNI failures at ...
      Host hostD is not functional or unable to check the status: <error message>
    Healthy hosts:  ['hostA', 'hostB', 'hostC']
    ```

    3.2. Get status of specific host

    example of healthy host:

    ```bash
    > python read_status.py hostA
    ===== Health Status of hostA =====
    Allocatable network devices: 2/2
    Connectable network devices: 2/2
    Host is OK (all functional and connected).
    Reported by checkerX at ...
    ```

    example of unhealthy host:

    ```bash
    > python read_status.py hostA
    ===== Health Status of hostD =====
    Allocatable network devices: 0/2
    Connectable network devices: 0/2
    Host hostD is not functional or unable to check the status: <error message>
    Reported by checkerX at ...
    ```
