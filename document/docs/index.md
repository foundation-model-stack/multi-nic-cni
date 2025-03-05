# Multi-NIC CNI

Multi-NIC CNI is the CNI plugin for secondary networks operating on top of [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni). This CNI offers several key features, outlined below, to help cluster administrators and users simplify the process of enabling high-performance networking.

- I) **Unifying user-managed network definition**: User can manage only one network definition for multiple secondary interfaces with a common CNI main plugin such as ipvlan, macvlan, and sr-iov.The Multi-NIC CNI automatically discovers all available secondary interfaces and handles them as a NIC pool.

  ![](./img/multi-nic-cni-feature-1.png)


-  II) **Bridging device plugin runtime results and CNI configuration:** Multi-NIC CNI can configure CNI of network device in accordance to device plugin allocation results orderly.

   ![](./img/multi-nic-cni-feature-2.png)

- III) **Building-in with several auto-configured CNIs**
Leveraging advantage point of managing multiple CNIs together with auto-discovery and dynamic interface selection, we built several auto-configured CNIs in the Multi-NIC CNI project including host-interface-local IPAM, multi-config IPAM, multi-gateway routing, and AWS-ipvlan CNI.

Check out the project on GitHub ➡️ [Multi-NIC CNI](https://github.com/foundation-model-stack/multi-nic-cni).

For more insights, please check

- [Medium Blog Post Series](https://medium.com/@sunyanan.choochotkaew1/list/multinic-cni-series-8570830e6f3f)
- [KubeCon+CloudNativeCon NA 20222 CNCF-Hosted Co-located Event](https://sched.co/1AsSs)
- [KubeCon+CloudNativeCon NA 20224 CNCF-Hosted Co-located Event](https://sched.co/1izs8)
- [Multi-NIC CNI in Vela IBM Research's AI supercomputer in the cloud](https://research.ibm.com/blog/openshift-foundation-model-stack)
- [The infrastructure powering IBM's Gen AI model development](https://arxiv.org/abs/2407.05467)