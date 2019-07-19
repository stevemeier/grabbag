#!/bin/bash

cd /root

export SSLMATE_APIKEY=get-your-own-jey

./certspotter --cert /usr/syno/etc/certificate/system/default/cert.pem

if [ $? -eq 0 ]; then
  ./cert-chain-resolver -o /usr/syno/etc/certificate/system/default/fullchain.pem /usr/syno/etc/certificate/system/default/cert.pem
  /usr/syno/sbin/synoservicectl --restart nginx
fi
