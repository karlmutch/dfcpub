#!/bin/bash
set -o xtrace
source /etc/profiled/dfcpaths.sh
sudo rm -rf /home/ubuntu/.dfc*
rm -rf /tmp/dfc*
cd $DFCSRC
for dfcpid in `ps -C dfc -o pid=`; do echo Stopping DFC $dfcpid; sudo kill $dfcpid; done
git pull
git status
git log | head -5

setup/deploy.sh -loglevel=3 -statstime=10s <<< $'4\n3\n2\n1'
ps -C dfc

echo create DFC local bucket
curl -i -X POST -H 'Content-Type: application/json' -d '{"action": "createlb"}' http://127.0.0.1:8080/v1/buckets/devTestLocal

echo run DFC tests with local bucket
BUCKET=devTestLocal go test -v -p 1 -count 1 -timeout 20m ./...

echo run DFC tests with cloud bucket
BUCKET=devtestcloud go test -v -p 1 -count 1 -timeout 20m ./...

echo delete DFC local bucket
curl -i -X DELETE -H 'Content-Type: application/json' -d '{"action": "destroylb"}' http://127.0.0.1:8080/v1/buckets/devTestLocal
