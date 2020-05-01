#!/bin/bash
set -e

scriptDir=$(cd `dirname $0` && pwd)

pushd "$scriptDir"
protoc -I service/api/refunction/v1alpha service/api/refunction/v1alpha/refunction.proto --go_out=plugins=grpc:service/api/refunction/v1alpha
popd
