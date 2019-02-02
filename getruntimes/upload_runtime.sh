#!/bin/bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: ./upload_runtime.sh runtime dockerTag"
	exit 1
fi

runtimeName=$1
dockerTag=$2

docker save "$dockerTag" > "$runtimeName.tar"

aws s3 cp "$runtimeName.tar" s3://refunction-runtimes

rm -rf "$runtimeName.tar"

