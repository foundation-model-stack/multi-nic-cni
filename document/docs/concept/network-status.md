# Network Status
Reported multinicnetwork status

- v1.0.2
  
  Field|Value|Description
  ---|---|---
  computeResults|netAddress, numOfHosts|summary of CIDR computation if `multiNICIPAM=true`
  routeStatus|Success|`mode=l3` and all routes are applied
  |WaitForRoutes|`mode=l3` but the new CIDR is just recomputed and waiting for route update
  |Failed|`mode=l3` but some route cannot be applied, need attention
  |Unknown|`mode=l3` but some daemon cannot be connected
  |N/A|`mode!=l3`
  lastSyncTime|Date Time|timestamp at last synchronization of interfaces and CIDR

- v1.0.3
  
  Field|Value|Description
  ---|---|---
  discovery|existDaemon,<br>infoAvailable,<br>cidrProcessed|results of hostinterface discovery
  computeResults|netAddress, numOfHosts|summary of CIDR computation if `multiNICIPAM=true`
  configStatus|Success<br>|network plugin has been configured.
  |WaitForConfig<br>|plugin configuration has not completed yet.
  |Failed|failed to configure network plugin
  routeStatus|Success|`mode=l3` and all routes are applied
  |WaitForRoutes|`mode=l3` but the new CIDR is just recomputed and waiting for route update
  |Failed|`mode=l3` but some route cannot be applied, need attention
  |Unknown|`mode=l3` but some daemon cannot be connected
  |N/A|`mode!=l3`
  message|ConfigError/RouteError|error message (if exists)
  lastSyncTime|Date Time|timestamp at last synchronization of interfaces and CIDR

