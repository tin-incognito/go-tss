#!/bin/sh

if [ "$1" == "validator-0" ]; then
echo 11234566 | ./cmd/tss/tss --tss_addr :4000 --p2p_port 4100 --log_level debug --bridge_signer_name validator0 --bridge_signer_password 11234566
fi
if [ "$1" == "validator-1" ]; then

while ! nc -z 127.0.0.1 4000; do
  echo sleeping
  sleep 1
done

echo 11234566 | ./cmd/tss/tss --tss_addr :4001 --peer /ip4/127.0.0.1/tcp/4100/ipfs/$(curl http://127.0.0.1:4000/p2pid) --p2p_port 4101 --log_level debug --bridge_signer_name validator1 --bridge_signer_password 11234566
fi
if [ "$1" == "validator-2" ]; then

while ! nc -z 127.0.0.1 4000; do
  echo sleeping
  sleep 1
done

echo 11234566 | ./cmd/tss/tss --tss_addr :4002 --peer /ip4/127.0.0.1/tcp/4100/ipfs/$(curl http://127.0.0.1:4000/p2pid) --p2p_port 4102 --log_level debug --bridge_signer_name validator2 --bridge_signer_password 11234566
fi
if [ "$1" == "validator-3" ]; then

while ! nc -z 127.0.0.1 4000; do
  echo sleeping
  sleep 1
done

echo 11234566 | ./cmd/tss/tss --tss_addr :4003 --peer /ip4/127.0.0.1/tcp/4100/ipfs/$(curl http://127.0.0.1:4000/p2pid) --p2p_port 4103 --log_level debug --bridge_signer_name validator3 --bridge_signer_password 11234566
fi
