# Supported CNI Plugins

The Multi-NIC CNI controller supports a limited set of common CNI plugins by parsing `.spec.plugin.args`.

Below is the list of supported CNIs and their corresponding arguments:

| CNI Type   | Supported Arguments                                      |
|------------|----------------------------------------------------------|
| `ipvlan`   | `master`, `mode`, `mtu`                                  |
| `macvlan`  | `master`, `mode`, `mtu`                                  |
| `awsipvlan`| `primaryIP`, `podIP`, `master`, `mode`, `mtu`            |
| `sriov`    | SriovNetworkNodePolicy:<br>`resourceName`, `priority`, `mtu`, `numVfs`, `isRdma`, `needVhostNet` <br><br>NetworkAttachmentDefinition:<br>`vlan`, `vlanQos`, `spoofchk`, `trust`, `min_tx_rate`, `max_tx_rate` |
| `mellanox` | *None*                                                   |

To add support for a new CNI plugin, please refer to [this example issue](https://github.com/foundation-model-stack/multi-nic-cni/issues/179).

Support must be implemented in the [plugin module](https://github.com/foundation-model-stack/multi-nic-cni/blob/main/internal/plugin) by adding a corresponding `GetConfig` function.
