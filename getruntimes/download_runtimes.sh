#!/bin/bash
set -euo pipefail

scriptdir=$(cd `dirname $0` && pwd)

declare -a runtimes=(
		      "alpine"
		      "python"
		      "alpinepython"
		      "java"
		      # "node"
		    )

for runtime in "${runtimes[@]}"
do
   wget https://s3.eu-west-2.amazonaws.com/refunction-runtimes/"$runtime".tar
   runtimeDir="$scriptdir/../worker/runtimes/$runtime"
   mkdir -p "$runtimeDir"
   tar -xf "$runtime.tar" -C "$runtimeDir"
   rm -rf "$runtime.tar"
done


