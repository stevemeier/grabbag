#!/bin/sh

VERSION=1.1.15
DOWNLOAD="https://github.com/sdeigm/uni-meter/releases/download/${VERSION}/uni-meter-${VERSION}.tgz"
TEMPDIR=$(mktemp -d)
WORKDIR="${TEMPDIR}/uni-meter"

mkdir -p ${WORKDIR}
mkdir -p ${WORKDIR}/DEBIAN
mkdir -p ${WORKDIR}/etc
mkdir -p ${WORKDIR}/usr/lib/systemd/system
mkdir -p ${WORKDIR}/usr/share/doc/uni-meter
mkdir -p ${WORKDIR}/opt/uni-meter

cat > ${WORKDIR}/DEBIAN/conffiles <<EOF
/etc/uni-meter.conf
EOF

cat > ${WORKDIR}/DEBIAN/control <<EOF
Package: uni-meter
Version: ${VERSION}
Architecture: all
Maintainer: sdeigm <sdeigm@github.com>
Section: java
Priority: optional
Depends: default-jre-headless, base-files (>= 12), base-files (<< 13)
Description: Universal electric meter data converter (emulator)
 Emulates an electrical meter like a Shelly Pro3EM or an EcoTracker 
EOF

cat > ${WORKDIR}/DEBIAN/postinst <<EOF
#!/bin/sh
set -e
systemctl daemon-reload
EOF
chmod 755 ${WORKDIR}/DEBIAN/postinst

cat > ${WORKDIR}/usr/share/doc/uni-meter/copyright <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Source: https://github.com/sdeigm/uni-meter

Files: *
Copyright: 2024, Stefan Deigmüller <sdeigm@github.com>
License: Apache-2
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 .
 http://www.apache.org/licenses/LICENSE-2.0
 .
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
Comment:
 On Debian systems, the complete text of the Apache License, Version 2.0
 can be found in "/usr/share/common-licenses/Apache-2.0".
EOF

cd ${WORKDIR}
curl -sL ${DOWNLOAD} | tar xzvf -
curl -sL https://raw.githubusercontent.com/sdeigm/uni-meter/refs/heads/main/CHANGELOG.md > ${WORKDIR}/usr/share/doc/uni-meter/changelog && gzip -9 ${WORKDIR}/usr/share/doc/uni-meter/changelog

mv ${WORKDIR}/uni-meter-${VERSION}/config/systemd/uni-meter.service ${WORKDIR}/usr/lib/systemd/system/
# Remove obsolete reference to syslog-target
sed -i 's/syslog.target //' ${WORKDIR}/usr/lib/systemd/system/uni-meter.service
rmdir ${WORKDIR}/uni-meter-${VERSION}/config/systemd

mv ${WORKDIR}/uni-meter-${VERSION}/config/uni-meter.conf ${WORKDIR}/etc/

mv ${WORKDIR}/uni-meter-${VERSION}/bin ${WORKDIR}/opt/uni-meter
mv ${WORKDIR}/uni-meter-${VERSION}/config ${WORKDIR}/opt/uni-meter
mv ${WORKDIR}/uni-meter-${VERSION}/lib ${WORKDIR}/opt/uni-meter

rmdir ${WORKDIR}/uni-meter-${VERSION}

cd ${TEMPDIR}
dpkg-deb --build uni-meter
if [ $? -eq 0 ]; then
  mv ${TEMPDIR}/uni-meter.deb ~
  echo '--- PACKAGE METADATA ---'
  dpkg -I ~/uni-meter.deb
  echo
  echo '--- PACKAGE CONTENT ---'
  dpkg -c ~/uni-meter.deb
  echo
fi

rm -rf ${TEMPDIR}
