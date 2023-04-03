#!/bin/sh
while ! nc -z 192.168.10.1 8080; do
  echo sleeping
  sleep 1
done

echo $SIGNER_PASSWORD | /go/bin/tss --tss_addr :8080 -peer /ip4/192.168.10.1/tcp/6668/ipfs/$(curl http://192.168.10.1:8080/p2pid) --p2p_port 6668 -log_level debug --bridge_signer_name $SIGNER_NAME --bridge_signer_password $SIGNER_PASSWORD
