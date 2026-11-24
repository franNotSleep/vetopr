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

# 1. check if firecracker is installed
if ! command -v firecracker > /dev/null 2>&1
then
  echo "Firecracker is not installed."
  echo "Installing firecracker..."

  rm -rf VETOPR_HOME

  mkdir -p $VETOPR_HOME/bin

  ARCH="$(uname -m)"
  release_url="https://github.com/firecracker-microvm/firecracker/releases"
  latest=$(basename $(curl -fsSLI -o /dev/null -w  %{url_effective} ${release_url}/latest))
  firecracker_folder_path=$(echo "release-${latest}-$(uname -m)")
  curl -L ${release_url}/download/${latest}/firecracker-${latest}-${ARCH}.tgz \
  | tar -xz

  # Rename the binary to "firecracker"
  mv $firecracker_folder_path/firecracker-${latest}-${ARCH} $FIRECRACKER_BIN
  rm -rf $firecracker_folder_path

  echo "export PATH="$HOME/vetopr/bin:$PATH"" >> ~/.bashrc
  echo "export PATH="$HOME/vetopr/bin:$PATH"" >> ~/.zshrc
fi

echo "Starting up server $API_SOCKET"
rm -f $API_SOCKET
firecracker --api-sock "${API_SOCKET}" --enable-pci &

sleep 0.5s

echo "Service started"

echo "Setting network..."

echo "Creating tap device $TAP"
sudo ip link del "$TAP" 2> /dev/null || true
if [ $? -ne 0 ]; then
  echo "unable to delete $TAP"
  exit
fi
sudo ip tuntap add $TAP mode tap user "$(id -u)" group "$(id -g)"
if [ $? -ne 0 ]; then
  echo sudo ip tuntap add $TAP mode tap user "$(id -u)" group "$(id -g)"
  exit
fi

echo "Assigning the TAP IP $TAP_IP and start up the device"
sudo ip addr add $TAP_IP/30 dev $TAP

sudo ip link set $TAP up
echo "Enabling IPv4 forwarding on the system"
sudo sysctl -w net.ipv4.ip_forward=1

echo "Setting up nftables & chains & rules"

if ! sudo nft list tables | grep -q "firecracker"; then
  sudo nft add table firecracker
  sudo nft 'add chain firecracker postrouting { type nat hook postrouting priority srcnat; policy accept; }'
  sudo nft 'add chain firecracker filter { type filter hook forward priority filter; policy accept; }'
  sudo nft add rule firecracker postrouting ip saddr $GUEST_IP oifname $HOST_IFACE counter masquerade
  sudo nft add rule firecracker filter iifname $TAP oifname $HOST_IFACE accept

  sudo iptables-nft -t nat -A POSTROUTING -o $HOST_IFACE -s $GUEST_IP -j MASQUERADE
  sudo iptables-nft -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
  sudo iptables-nft -A FORWARD -i $TAP -o $HOST_IFACE -j ACCEPT

fi


sudo curl -X PUT --unix-socket "${API_SOCKET}" \
    --data "{
        \"kernel_image_path\": \"${KERNEL}\",
        \"boot_args\": \"${KERNEL_BOOT_ARGS}\"
    }" \
    "http://localhost/boot-source"

sudo curl -X PUT --unix-socket "${API_SOCKET}" \
    --data "{
        \"drive_id\": \"rootfs\",
        \"path_on_host\": \"${ROOTFS}\",
        \"is_root_device\": true,
        \"is_read_only\": false
    }" \
    "http://localhost/drives/rootfs"

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

sleep 2s

# Setup internet access in the guest
ssh -i $KEY_NAME root@$GUEST_IP  "ip route add default via 172.16.0.1 dev eth0"

# Setup DNS resolution in the guest
ssh -i $KEY_NAME root@$GUEST_IP  "echo 'nameserver 8.8.8.8' > /etc/resolv.conf"

# glibc resolves A/AAAA in parallel; on an IPv6-connected host the AAAA query
# can stall on host NAT, adding seconds per lookup.
ssh -i $KEY_NAME root@$GUEST_IP  "echo 'options single-request-reopen' >> /etc/resolv.conf"

