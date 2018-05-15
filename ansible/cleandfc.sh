set -o
echo "Cleaning up targets"
parallel-ssh -h inventory/targets.txt -i './cleandfcstate.sh'
echo "Cleaning up proxy"
parallel-ssh -h inventory/proxy.txt -i './cleandfcstate.sh'
