#!/bin/sh

if [ "$1" == "validator-0" ]; then
echo ZThiMDAxOTk2MDc4ODk3YWE0YThlMjdkMWY0NjA1MTAwZDgyNDkyYzdhNmMwZWQ3MDBhMWIyMjNmNGMzYjVhYg== | ./cmd/tss/tss -tss-port :4000 -p2p-port 4100 -loglevel debug -signer_name validator0 -signer_password validator0_password 
fi
if [ "$1" == "validator-1" ]; then

while ! nc -z 127.0.0.1 4000; do
  echo sleeping
  sleep 1
done

echo ZTc2ZjI5OTIwOGVlMDk2N2M3Yzc1MjYyODQ0OGUyMjE3NGJiOGRmNGQyZmVmODg0NzQwNmUzYTk1YmQyODlmNA== | ./cmd/tss/tss -tss-port :4001 -peer /ip4/127.0.0.1/tcp/4100/ipfs/$(curl http://127.0.0.1:4000/p2pid) -p2p-port 4101 -loglevel debug -signer_name validator1 -signer_password validator1_paswword
fi
if [ "$1" == "validator-2" ]; then

while ! nc -z 127.0.0.1 4000; do
  echo sleeping
  sleep 1
done

echo MjQ1MDc2MmM4MjU5YjRhZjhhNmFjMmI0ZDBkNzBkOGE1ZTBmNDQ5NGI4NzM4OTYyM2E3MmI0OWMzNmE1ODZhNw== | ./cmd/tss/tss -tss-port :4002 -peer /ip4/127.0.0.1/tcp/4100/ipfs/$(curl http://127.0.0.1:4000/p2pid) -p2p-port 4102 -loglevel debug -signer_name validator2 -signer_password validator2_paswword
fi
if [ "$1" == "validator-3" ]; then

while ! nc -z 127.0.0.1 4000; do
  echo sleeping
  sleep 1
done

echo YmNiMzA2ODU1NWNjMzk3NDE1OWMwMTM3MDU0NTNjN2YwMzYzZmVhZDE5NmU3NzRhOTMwOWIxN2QyZTQ0MzdkNg== | ./cmd/tss/tss -tss-port :4003 -peer /ip4/127.0.0.1/tcp/4100/ipfs/$(curl http://127.0.0.1:4000/p2pid) -p2p-port 4103 -loglevel debug -signer_name validator3 -signer_password validator3_paswword
fi
