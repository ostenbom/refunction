#!/bin/bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: makelayer.sh imagename"
  exit 1
fi
imagename=$1

cd ../images/"$imagename"
make
cd -

mkdir -p "$imagename"/bin
cp ../images/"$imagename"/"$imagename" "$imagename"/bin/"$imagename"
cd "$imagename"
tar -cvf layer.tar .
cd ..
rm -rf "${imagename:?}"/bin
