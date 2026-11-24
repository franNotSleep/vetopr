#! /usr/bin/bash

VETOPR_HOME="$HOME/vetopr"
FIRECRACKER_BIN="$VETOPR_HOME/bin/firecracker"
API_SOCKET="/tmp/firecracker-$1.socket"
TAP=$2
TAP_IP=$3
GUEST_IP=$4
HOST_IFACE=$(ip -j route list default | jq -r '.[0].dev')
MAC=$5
KERNEL="./$(ls ./artifacts/vmlinux* | tail -1)"
KERNEL_BOOT_ARGS="console=ttyS0 reboot=k panic=1"
ROOTFS="./$(ls ./artifacts/*.ext4 | tail -1)"
KEY_NAME=./$(ls ./artifacts/*.id_rsa | tail -1)


echo "Running directory: $PWD"

if  [ ! -f $KERNEL ]; then
  echo "Kernel file is required"
  exit 1
fi

if  [ ! -f $ROOTFS ]; then
  echo "filesystem file is required"
  exit 1
fi

echo "Starting up server $API_SOCKET"
rm -f $API_SOCKET
firecracker --api-sock "${API_SOCKET}" --enable-pci
