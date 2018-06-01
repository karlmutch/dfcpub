#!/bin/bash

DFCDIR=$1
RUNTEST=$2
BRANCH=$3
CLUSTER=$4

source /home/ubuntu/aws.env 
mkdir $DFCDIR
echo 'Cluster DFCDIR' 
echo $DFCDIR
cd $DFCDIR
git clone https://github.com/NVIDIA/dfcpub.git
cd dfcpub
git checkout dfc_ansible
cd ansible
pwd
ls -al   
python -u aws_cluster.py --command restart --cluster $CLUSTER
if $RUNTEST; then
    echo running devtest on branch $BRANCH
    parallel-scp -h inventory/targets.txt rundevtest.sh '/home/ubuntu/'
    ssh $(head -1 inventory/targets.txt) './rundevtest.sh $BRANCH'
    EXIT_STATUS=$?
    echo RUNTEST exit status is $EXIT_STATUS
fi
if $SHUTDOWN; then
	echo shutting down cluster
	python -u aws_cluster.py --command shutdown --cluster $CLUSTER
fi
exit $EXIT_STATUS
