API_SOCKET="/tmp/firecracker.socket"

start-firecracker:
	sudo rm -f $(API_SOCKET)
	sudo ./firecracker --api-sock "${API_SOCKET}" --enable-pci

