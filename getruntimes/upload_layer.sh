#!/bin/bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: ./upload_layer.sh layerName"
	exit 1
fi

layerName=$1
scriptDir=$(cd `dirname $0` && pwd)

aws s3 cp --recursive "$scriptDir/../worker/activelayers/$layerName" s3://refunction-runtimes/layers/"$layerName"

