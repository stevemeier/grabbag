#!/bin/sh

export SPACEWALK_USER=admin
export SPACEWALK_PASS=admin1

cd 

/usr/bin/spacewalk-repo-sync -c centos6-i386-updates
./errata-import.pl --server localhost --errata errata.latest.xml --publish --debug | tee import-1.log
/usr/bin/spacewalk-repo-sync -c centos6-x86_64-updates
./errata-import.pl --server localhost --errata errata.latest.xml --publish --debug | tee import-2.log
/usr/bin/spacewalk-repo-sync -c centos7-x86_64-updates
./errata-import.pl --server localhost --errata errata.latest.xml --publish --debug | tee import-3.log
