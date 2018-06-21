#!/bin/bash
set -e
echo Creating disks total of $#

chmod 744 createdfcvolumes.sh
./createdfcvolumes.sh $#

for disk in "$@"; do
    sudo mkdir -p /dfc/$disk || true
    sudo mkfs -t xfs /dev/$disk || true
done
