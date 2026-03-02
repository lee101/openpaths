#!/bin/bash
set -e

CUDA_PATH="${CUDA_PATH:-/usr/local/cuda-12.9}"
GOBED_GPU="${GOBED_GPU:-/home/lee/code/gobed/gpu}"

export LD_LIBRARY_PATH="${GOBED_GPU}:${CUDA_PATH}/lib64:${LD_LIBRARY_PATH}"
export GOMAXPROCS="${GOMAXPROCS:-3}"

exec ./bin/openpaths-gpu "$@"
