#!/bin/bash
set -o
echo "Cleaning up targets"
parallel-ssh -h inventory/targets.txt -i 'ls /dfc'
parallel-ssh -h inventory/targets.txt -i 'sudo rm -rf /var/log/dfc*'
parallel-ssh -h inventory/targets.txt -i 'sudo rm -rf /dfc/localbuckets'
parallel-ssh -h inventory/targets.txt -i 'sudo rm -rf /dfc/mpaths'
parallel-ssh -t 180 -h inventory/targets.txt -i 'sudo rm -rf /dfc/*/*'
parallel-ssh -h inventory/targets.txt -i 'ls /dfc'
parallel-ssh -h inventory/targets.txt -i 'df -h'
echo "Cleaning up proxy"
parallel-ssh -h inventory/proxy.txt -i 'sudo rm -rf /var/log/dfc*'
parallel-ssh -h inventory/proxy.txt -i 'sudo rm -rf /dfc/*'
parallel-ssh -h inventory/proxy.txt -i 'ls /dfc'
