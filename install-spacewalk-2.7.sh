#!/bin/bash

# Temporary directory for downloads
TEMPDIR="/tmp"

# curl options
CURLOPTS="-s -m 5"

# Identify CentOS Version
CENTOSVERSION=`rpm -q centos-release --qf '%{VERSION}' 2>/dev/null`

# Repository URL
REPOURL="http://yum.spacewalkproject.org"

# Check minimum CentOS Version
if [ ${CENTOSVERSION} -lt 6 ]; then
  echo "ERROR: Sorry, this CentOS Version is not supported"
  exit 1
fi

# Import Spacewalk GPG KEY
export KEYYEAR=2015
if [ ! -f /etc/pki/rpm-gpg/RPM-GPG-KEY-spacewalk-${KEYYEAR} ]; then 
  echo
  echo "##################################"
  echo "## Installing Spacewalk GPG Key ##"
  echo "##################################"
  (cd /etc/pki/rpm-gpg && curl ${CURLOPTS} -O ${REPOURL}/RPM-GPG-KEY-spacewalk-${KEYYEAR})
  rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-spacewalk-${KEYYEAR}
fi


# Install Spacewalk repo
if [ ! -f /etc/yum.repos.d/spacewalk.repo ]; then
  echo
  echo "#####################################"
  echo "## Installing Spacewalk Repository ##"
  echo "#####################################"
  (cd ${TEMPDIR} && curl ${CURLOPTS} -O ${REPOURL}/2.7/RHEL/${CENTOSVERSION}/x86_64/spacewalk-repo-2.7-2.el${CENTOSVERSION}.noarch.rpm)
  rpm -Uvh ${TEMPDIR}/spacewalk-repo-*.noarch.rpm || exit 1
  rm -f ${TEMPDIR}/spacewalk-repo-*.noarch.rpm
fi

# Install EPEL (for dependencies)
if [ ! -f /etc/yum.repos.d/epel.repo ]; then
  echo
  echo "################################"
  echo "## Installing EPEL Repository ##"
  echo "################################"
  if [ "${CENTOSVERSION}" -eq 6 ]; then
    (cd ${TEMPDIR} && curl ${CURLOPTS} -O https://dl.fedoraproject.org/pub/epel/epel-release-latest-6.noarch.rpm)
  fi
  if [ "${CENTOSVERSION}" -eq 7 ]; then
    (cd ${TEMPDIR} && curl ${CURLOPTS} -O https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm)
  fi
  rpm -Uvh ${TEMPDIR}/epel-release-*.noarch.rpm || exit 1
  rm -f ${TEMPDIR}/epel-release-*.noarch.rpm 
fi

# Spacewalk Client Tools (for dependencies, too)
if [ ! -f /etc/yum.repos.d/spacewalk-client.repo ]; then
  echo
  echo "############################################"
  echo "## Installing Spacewalk-Client Repository ##"
  echo "############################################"
  (cd ${TEMPDIR} && curl ${CURLOPTS} -O ${REPOURL}/2.7/RHEL/${CENTOSVERSION}/x86_64/spacewalk-client-repo-2.7-2.el${CENTOSVERSION}.noarch.rpm)
  rpm -ihv ${TEMPDIR}/spacewalk-client-repo-*.noarch.rpm || exit 1
  rm -f ${TEMPDIR}/spacewalk-client-repo-*.noarch.rpm
fi

# Add Java repository
cd /etc/yum.repos.d
if [ "${CENTOSVERSION}" -eq 6 ]; then
  curl ${CURLOPTS} -O https://copr.fedorainfracloud.org/coprs/g/spacewalkproject/java-packages/repo/epel-7/group_spacewalkproject-java-packages-epel-7.repo
  curl ${CURLOPTS} -O https://copr.fedorainfracloud.org/coprs/g/spacewalkproject/epel6-addons/repo/epel-6/group_spacewalkproject-epel6-addons-epel-6.repo
fi
if [ "${CENTOSVERSION}" -eq 7 ]; then
  curl ${CURLOPTS} -O https://copr.fedorainfracloud.org/coprs/g/spacewalkproject/java-packages/repo/epel-7/group_spacewalkproject-java-packages-epel-7.repo
fi

# Import RED HAT GPG Key
if [ ! -f /etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release ]; then 
  echo
  echo "################################"
  echo "## Installing Red Hat GPG Key ##"
  echo "################################"
  (cd /etc/pki/rpm-gpg && curl ${CURLOPTS} http://www.redhat.com/security/37017186.txt > /etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release)
  rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
fi

# Install Spacewalk
rpm -qi spacewalk-common 2>&1 >/dev/null
if [ $? -eq 1 ]; then
  echo
  echo "##########################"
  echo "## Installing Spacewalk ##"
  echo "##########################"
  yum --nogpgcheck -y install spacewalk-postgresql spacewalk-setup-postgresql spacecmd
fi

# Create answer file
ANSWERFILE=`mktemp`
cat > $ANSWERFILE <<EOF
admin-email = root@localhost
ssl-set-org = .
ssl-set-org-unit = .
ssl-set-city = .
ssl-set-state = .
ssl-set-country = DE
ssl-password = spacewalk
ssl-set-email = root@localhost
ssl-set-cnames = .
ssl-config-sslvhost = Y
db-backend=postgresql
db-name=spaceschema
db-user=spaceuser
db-password=supersecret
db-host=localhost
db-port=5432
enable-tftp=Y
EOF

# Add firewall rules
if [ ${CENTOSVERSION} -eq 7 ]; then
  if [ -x /usr/bin/firewall-cmd ]; then
    echo "#################################"
    echo "## Opening firewall for HTTP/S ##"
    echo "#################################"
    firewall-cmd -q --add-service=http
    firewall-cmd -q --add-service=https
  fi
fi

# Run setup
grep db_password /etc/rhn/rhn.conf > /dev/null
if [ $? -eq 1 ]; then
  echo
  echo "##########################"
  echo "## Setting up Spacewalk ##"
  echo "##########################"
  spacewalk-setup --answer-file=${ANSWERFILE}
fi
rm -f ${ANSWERFILE}

# Run pgtune
echo
echo "####################"
echo "## Running pgtune ##"
echo "####################"
yum -y install pgtune

pgtune --type=web -c 600 -i /var/lib/pgsql/data/postgresql.conf > /var/lib/pgsql/data/postgresql.conf.pgtune
mv /var/lib/pgsql/data/postgresql.conf /var/lib/pgsql/data/postgresql.conf.orig
cd /var/lib/pgsql/data/
ln -s postgresql.conf.pgtune postgresql.conf

service postgresql restart

# Get ISOs for CentOS 6 and 7
echo
echo "#############################"
echo "## Downloading CentOS ISOs ##"
echo "#############################"
mkdir -p /var/iso-images/
mkdir -p /var/distro-trees/CentOS-6-x86_64
mkdir -p /var/distro-trees/CentOS-7-x86_64
cd /var/iso-images
curl ${CURLOPTS} -O http://mirror.rackspace.com/CentOS/6/isos/x86_64/CentOS-6.9-x86_64-netinstall.iso
curl ${CURLOPTS} -O http://mirror.rackspace.com/CentOS/7/isos/x86_64/CentOS-7-x86_64-NetInstall-1708.iso

cat >> /etc/fstab <<EOF
/var/iso-images/CentOS-6.9-x86_64-netinstall.iso           /var/distro-trees/CentOS-6-x86_64       iso9660 loop,ro 0 0
/var/iso-images/CentOS-7-x86_64-NetInstall-1708.iso        /var/distro-trees/CentOS-7-x86_64       iso9660 loop,ro 0 0
EOF

echo
echo "##########################"
echo "## Mounting CentOS ISOs ##"
echo "##########################"
mount /var/distro-trees/CentOS-6-x86_64
mount /var/distro-trees/CentOS-7-x86_64

echo
echo "============================="
echo "=== INSTALLATION COMPLETE ==="
echo "============================="
echo
