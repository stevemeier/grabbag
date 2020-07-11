#!/bin/bash

for I in `seq 0 255`
do
	echo "### $I.in-addr.arpa ###"
	dig +short @8.8.8.8 $I.in-addr.arpa ns | tr '[:upper:]' '[:lower:]' | rev | sort | rev | sed -e 's/^/\`/g' -e 's/$/\`/g'
	echo
done
