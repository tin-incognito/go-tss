#!/bin/sh
echo $SIGNER_PASSWORD | /go/bin/tss --tss_addr :8080  --p2p_port 6668 -loglevel debug --bridge_signer_name $SIGNER_NAME --bridge_signer_password $SIGNER_PASSWORD