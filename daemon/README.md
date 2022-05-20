## Multi-NIC CNI Daemon

The daemon component of Multi-NIC CNI is deployed to the Kubernetes cluster as a DaemonSet. This component is for 
- discovering host interfaces (/interface)
- configuring host L3 routes (/addroute, /deleteroute)
- distributedly communicating with Multi-NIC CNI binaries
    - select NICs (/select)
    - allocate/deallocate IPs (/allocate, /deallocate)

The default Daemon port is 11000. However, it can be set by setting `` environment.
### Build and Push Daemon
1. Update IMAGE_REGISTRY, IMAGE_TAG_BASE, and DAEMON_VERSION in Makefile if needed
2. Run
    ```bash
    make docker-build-push
    ```
    To build with test, run 
    ```bash
    make docker-build-push-with-test
    ```