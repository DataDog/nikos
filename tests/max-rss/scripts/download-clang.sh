#!/bin/bash

arch_mapping=( ["aarch64"]="arm64" ["arm64"]="arm64" ["x86_64"]:"amd64" ["amd64"]:"amd64" ["x64"]:"amd64" )

MACHINE=$(uname -m)
ARCH="${arch_mapping[${MACHINE}]}"

CLANG_URL="https://dd-agent-omnibus.s3.amazonaws.com/llvm/clang-14.0.5.${ARCH}"
wget $CLANG_URL -O ./tools/clang-14
chmod 0755 ./tools/clang-14
