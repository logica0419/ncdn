#!/bin/bash

sudo apt update
sudo apt install --no-install-recommends -y \
    libssl-dev sshpass dnsutils ethtool \
    supervisor iputils-ping tcpdump bind9-dnsutils \
    build-essential libbpf-dev clang llvm \
    flex bison dwarves libelf-dev bc

git clone https://github.com/microsoft/WSL2-Linux-Kernel \
  -b linux-msft-wsl-$(uname -r | cut -d"-" -f 1) --single-branch --depth=1
cd WSL2-Linux-Kernel
echo 3 | make -j$(nproc) KCONFIG_CONFIG=./Microsoft/config-wsl
cd ..
rm -rf WSL2-Linux-Kernel

go install github.com/cespare/reflex@latest
