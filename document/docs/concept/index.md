# Key Concept

The key concept of Multi-NIC CNI is to provide network segmentation and top-up network bandwidth in the containerization system by attaching secondary network interfaces that is linked to different network interfaces on host (NIC) to pod in a simple, adaptive, and scaled manner.

The core features of Multi-NIC CNI are:

- **Common secondary network definition**: User can manage only one network definition for multiple secondary interfaces with a common CNI main plugin such as ipvlan, macvlan, and sr-iov. 

- **Common NAT-bypassing network solution**: All secondary NICs on each host can be assigned with non-conflict CIDR and non-conflict L3 routing configuration that can omit an overlay networking overhead. Particularyly, the CNI is built-in with L3 IPVLAN solution composing of the following functionalities.
    * **Interface-host-devision CIDR Computation**: compute allocating CIDR range for each host and each interface from a single global subnet with the number of bits for hosts and for interface. 
    * **L3 Host Route Configuration**: configure L3 routes (next hop via dev) in host route table according to the computed CIDR.
    * **Distributed IP Allocation Management**: manage IP allocation/deallocation distributedly via the communication between CNI program and daemon at each host.

- **Policy-based secondary network attachment**: Instead of statically set the desired host's master interface name one by one, user can define a policy on attaching multiple secondary network interfaces such as specifying only the number of desired interfaces, filtering only highspeed NICs. 

The Multi-NIC operator operates over a custom resource named *MultiNicNetwork* defined by users. This definition will define a Pod global subnet, common network definition (main CNI and IPAM plugin), and attachment policy. 

After deploying *MultiNicNetwork*, *NetworkAttachmentDefinition* with the same name will be automatically configured and created respectively.