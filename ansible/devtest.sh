#!/bin/bash
cd $1
python -u aws_cluster.py --command restart --cluster Cluster3
echo running devtest on branch $BRANCH
parallel-scp -h inventory/targets.txt rundevtest.sh '/home/ubuntu/'
ssh $(head -1 inventory/targets.txt) './rundevtest.sh master'
EXIT_STATUS=$?
echo RUNTEST exit status is $EXIT_STATUS
exit $EXIT_STATUS
