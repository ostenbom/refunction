#!/bin/bash
set -euo pipefail

scriptDir=$(cd `dirname $0` && pwd)

pushd "$scriptDir"
go build .
aws s3 cp cri s3://refunction-cri/cri --acl public-read
popd

pushd "$scriptDir/../funk"
go build .
aws s3 cp funk s3://refunction-cri/funk --acl public-read
popd
