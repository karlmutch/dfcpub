#!/bin/bash
set -e
for disk in "$@"; do
    sudo mkdir -p /dfc/$disk || true
    sudo mkfs -t xfs /dev/$disk || true
done
