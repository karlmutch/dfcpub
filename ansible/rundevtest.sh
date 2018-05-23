#!/bin/bash
set -o xtrace
source /etc/profile.d/dfcpaths.sh
for dfcpid in `ps -C dfc -o pid=`; do echo Stopping DFC $dfcpid; sudo kill $dfcpid; done
sudo rm -rf /home/ubuntu/.dfc*
rm -rf /tmp/dfc*
cd $DFCSRC
if [ ! -z $1 ]; then
    echo Git checkout branch $1
    git checkout $1
fi

git pull
git status
git log | head -5

setup/deploy.sh -loglevel=3 -statstime=10s <<< $'4\n3\n2\n1'

echo sleep 10 seconds before checking DFC process
sleep 10
dfcprocs=$(ps -C dfc -o pid= | wc -l)
echo number of dfcprocs $dfcprocs
if [ $dfcprocs -lt 7 ]; then
    echo dfc did not start properly
    exit 1
fi
echo create DFC local bucket
curl -i -X POST -H 'Content-Type: application/json' -d '{"action": "createlb"}' http://127.0.0.1:8080/v1/buckets/devTestLocal

echo run DFC tests with local bucket
BUCKET=devTestLocal go test -v -p 1 -count 1 -timeout 20m ./...

echo run DFC tests with cloud bucket
BUCKET=devtestcloud go test -v -p 1 -count 1 -timeout 20m ./...

echo delete DFC local bucket
curl -i -X DELETE -H 'Content-Type: application/json' -d '{"action": "destroylb"}' http://127.0.0.1:8080/v1/buckets/devTestLocal

for dfcpid in `ps -C dfc -o pid=`; do echo Stopping DFC $dfcpid; sudo kill $dfcpid; done
