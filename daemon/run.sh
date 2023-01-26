#!/bin/bash
cp /usr/local/app/multi-nic /host/opt/cni/bin/multi-nic
cp /usr/local/app/multi-nic-ipam /host/opt/cni/bin/multi-nic-ipam
cp /usr/local/app/aws-ipvlan /host/opt/cni/bin/aws-ipvlan
./daemon