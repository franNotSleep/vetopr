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
KERNEL_BOOT_ARGS="console=ttyS0 reboot=k panic=1 ip=$GUEST_IP::$TAP_IP:255.255.255.252::eth0:off"
ROOTFS="./$(ls ./artifacts/*.ext4 | tail -1)"
KEY_NAME=./$(ls ./artifacts/*.id_rsa | tail -1)

echo "Service started"

echo "Setting network..."

echo "Creating tap device $TAP"
sudo ip link del "$TAP" 2> /dev/null || true
sudo ip tuntap add $TAP mode tap user "$(id -u)" group "$(id -g)"
echo "Assigning the TAP IP $TAP_IP and start up the device"
sudo ip addr add $TAP_IP/30 dev $TAP
sudo ip link set $TAP up
echo "Enabling IPv4 forwarding on the system"
sudo sysctl -w net.ipv4.ip_forward=1


if ! sudo nft list tables | grep -q "firecracker"; then
  echo "Setting up tables & chains"
  sudo nft add table firecracker
  sudo nft 'add chain firecracker postrouting { type nat hook postrouting priority srcnat; policy accept; }'
  sudo nft 'add chain firecracker filter { type filter hook forward priority filter; policy accept; }'
fi

echo "Setting up rules"
sudo nft add rule firecracker postrouting ip saddr $GUEST_IP oifname $HOST_IFACE counter masquerade
sudo nft add rule firecracker filter iifname $TAP oifname $HOST_IFACE accept

sudo iptables-nft -t nat -A POSTROUTING -o eth0 -s $GUEST_IP -j MASQUERADE
sudo iptables-nft -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
sudo iptables-nft -A FORWARD -i $TAP -o $HOST_IFACE -j ACCEPT

echo "Setting up boot source"
sudo curl -X PUT --unix-socket "${API_SOCKET}" \
    --data "{
        \"kernel_image_path\": \"${KERNEL}\",
        \"boot_args\": \"${KERNEL_BOOT_ARGS}\"
    }" \
    "http://localhost/boot-source"
if [ $? -ne 0 ]; then
  exit 1
fi
if [ $? -ne 0 ]; then
  echo "SETTING FILESYSTEM"
  exit 1
fi

sudo curl -X PUT --unix-socket "${API_SOCKET}" \
    --data "{
        \"drive_id\": \"rootfs\",
        \"path_on_host\": \"${ROOTFS}\",
        \"is_root_device\": true,
        \"is_read_only\": false
    }" \
    "http://localhost/drives/rootfs"
if [ $? -ne 0 ]; then
  echo "SETTING FILESYSTEM"
  exit 1
fi

curl --unix-socket "${API_SOCKET}" -i \
  -X PUT 'http://localhost/network-interfaces/net1' \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json' \
  -d "{
      \"iface_id\": \"net1\",
      \"guest_mac\": \"$MAC\",
      \"host_dev_name\": \"$TAP\"
    }"


sleep 0.015s

sudo curl -X PUT --unix-socket "${API_SOCKET}" \
    --data "{
        \"action_type\": \"InstanceStart\"
    }" \
    "http://localhost/actions"
if [ $? -ne 0 ]; then
  echo "SETTING FILESYSTEM"
  exit 1
fi
