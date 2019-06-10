#!/bin/bash

cd /root

export CENSYS_APIID=get-your
export CENSYS_SECRET=own-key

./censys --cert /usr/syno/etc/certificate/system/default/cert.pem \
         --fullchain /usr/syno/etc/certificate/system/default/fullchain.pem

if [ $? -eq 0 ]; then
  /usr/syno/sbin/synoservicectl --restart nginx
fi
