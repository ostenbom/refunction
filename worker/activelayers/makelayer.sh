#!/bin/bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: makelayer.sh imagename"
  exit 1
fi
imagename=$1
scriptdir=$(cd `dirname $0` && pwd)

if [ -f "$scriptdir"/../images/"$imagename"/Makefile ]; then
  pushd "$scriptdir"/../images/"$imagename"
  make
  popd
fi

mkdir -p "$scriptdir"/"$imagename"

for f in "$scriptdir"/../images/"$imagename"/*; do
  if [[ "$f" =~ .*\....?.?.?$ ]]; then
    cp "$f" "$scriptdir"/"$imagename"
  else
    mkdir -p "$scriptdir"/"$imagename"/bin
    cp "$f" "$scriptdir"/"$imagename"/bin/
  fi
done

pushd "$scriptdir"/"$imagename"
tar -cvf layer.tar .
find . ! -name layer.tar -exec rm -rf {} +
popd
