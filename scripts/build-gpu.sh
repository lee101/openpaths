#!/bin/bash
set -e

CUDA_PATH="${CUDA_PATH:-/usr/local/cuda-12.9}"
GOBED_GPU="${GOBED_GPU:-/home/lee/code/gobed/gpu}"

export CGO_CFLAGS="-I${GOBED_GPU}"
export CGO_LDFLAGS="-L${GOBED_GPU} -L${CUDA_PATH}/lib64"
export LD_LIBRARY_PATH="${GOBED_GPU}:${CUDA_PATH}/lib64:${LD_LIBRARY_PATH}"
export GOMAXPROCS="${GOMAXPROCS:-3}"

go build -tags="gpu cuda" \
  -ldflags="-extldflags '-Wl,-rpath,${GOBED_GPU} -Wl,-rpath,${CUDA_PATH}/lib64'" \
  -o bin/openpath-gpu ./cmd/openpath/

echo "Built bin/openpath-gpu (CUDA ${CUDA_PATH})"
