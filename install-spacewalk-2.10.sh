#!/bin/bash

# Sync Channels?
if [ -z "${SYNC}" ]; then
  SYNC=0
fi

# Temporary directory for downloads
TEMPDIR="/tmp"

# curl options
CURLOPTS="-s -m 5"

# Identify CentOS Version
CENTOSVERSION=`rpm -q centos-release --qf '%{VERSION}' 2>/dev/null`

# Check minimum CentOS Version
if [ ${CENTOSVERSION} -lt 6 ]; then
  echo "ERROR: Sorry, this CentOS Version is not supported"
  exit 1
fi

# Install Spacewalk repo
if [ ! -f /etc/yum.repos.d/spacewalk.repo ]; then
  echo
  echo "#####################################"
  echo "## Installing Spacewalk Repository ##"
  echo "#####################################"
  yum install -y yum-plugin-tmprepo
  if [ "${CENTOSVERSION}" -eq 6 ]; then
    yum install -y spacewalk-repo --tmprepo=https://copr-be.cloud.fedoraproject.org/results/%40spacewalkproject/spacewalk-2.10/epel-6-x86_64/repodata/repomd.xml --nogpg
  fi
  if [ "${CENTOSVERSION}" -eq 7 ]; then
    yum install -y spacewalk-repo --tmprepo=https://copr-be.cloud.fedoraproject.org/results/%40spacewalkproject/spacewalk-2.10/epel-7-x86_64/repodata/repomd.xml --nogpg
  fi
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

# Install Spacewalk
rpm -qi spacewalk-common 2>&1 >/dev/null
if [ $? -eq 1 ]; then
  echo
  echo "##########################"
  echo "## Installing Spacewalk ##"
  echo "##########################"
  yum --nogpgcheck -y install spacewalk-postgresql spacewalk-setup-postgresql spacecmd spacewalk-utils
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
curl ${CURLOPTS} -O http://mirror.rackspace.com/CentOS/6/isos/x86_64/CentOS-6.10-x86_64-netinstall.iso
curl ${CURLOPTS} -O http://mirror.rackspace.com/CentOS/7/isos/x86_64/CentOS-7-x86_64-NetInstall-2003.iso

cat >> /etc/fstab <<EOF
/var/iso-images/CentOS-6.10-x86_64-netinstall.iso           /var/distro-trees/CentOS-6-x86_64       iso9660 loop,ro 0 0
/var/iso-images/CentOS-7-x86_64-NetInstall-2003.iso         /var/distro-trees/CentOS-7-x86_64       iso9660 loop,ro 0 0
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
echo "###########################"
echo "## Setting up admin user ##"
echo "###########################"
echo
SWUSER=admin
SWPASS=admin1
TEMPFILE=$(mktemp)

# from https://gist.github.com/vinzent/4bba600573bc9eeb33c4#gistcomment-1810454
# Since Spacewalk 2.9 (or 2.8?) satwho now produces one line if there are no users
if [ "$(satwho | wc -l)" = "1" ]; then
  echo "INFO: Creating user admin"
  curl --silent https://localhost/rhn/newlogin/CreateFirstUser.do --insecure -D - >${TEMPFILE}

  cookie=$(egrep -o 'JSESSIONID=[^ ]+' ${TEMPFILE})
  csrf=$(egrep csrf_token ${TEMPFILE} | egrep -o 'value=[^ ]+' | egrep -o '[0-9]+')

    curl --noproxy '*' \
      --cookie "$cookie" \
      --insecure \
      --data "csrf_token=-${csrf}&submitted=true&orgName=DefaultOrganization&login=${SWUSER}&desiredpassword=${SWPASS}&desiredpasswordConfirm=${SWPASS}&email=root%40localhost&prefix=Mr.&firstNames=Administrator&lastName=Spacewalk&" \
      https://localhost/rhn/newlogin/CreateFirstUser.do

  if [ "$(satwho | wc -l)" = "1" ]; then
    echo "ERROR: User creation failed" >&2
  fi
fi
rm -f ${TEMPFILE}

echo
echo "#########################"
echo "## Setting up channels ##"
echo "#########################"
echo
/usr/bin/spacewalk-common-channels -v -u ${SWUSER} -p ${SWPASS} -a i386,x86_64 'centos6*'
/usr/bin/spacewalk-common-channels -v -u ${SWUSER} -p ${SWPASS} -a x86_64 'centos7*'

if [ ${SYNC} -gt 0 ]; then
  echo
  echo "#######################################"
  echo "## Starting Sync for Update channels ##"
  echo "#######################################"
  echo
  /usr/bin/spacecmd -u ${SWUSER} -p ${SWPASS} softwarechannel_syncrepos centos6-i386-updates
  /usr/bin/spacecmd -u ${SWUSER} -p ${SWPASS} softwarechannel_syncrepos centos6-x86_64-updates
  /usr/bin/spacecmd -u ${SWUSER} -p ${SWPASS} softwarechannel_syncrepos centos7-x86_64-updates
fi

echo 
echo "################################"
echo "## Installing useful packages ##"
echo "################################"
echo
yum -y install wget perl-Frontier-RPC perl-Text-Unidecode 
cd
wget https://cefs.steve-meier.de/errata.latest.xml
wget https://www.redhat.com/security/data/oval/com.redhat.rhsa-all.xml
wget https://cefs.steve-meier.de/errata-import.tar
tar xf errata-import.tar

echo To test CEFS run:
echo export SPACEWALK_USER=${SWUSER} SPACEWALK_PASS=${SWPASS}
echo ./errata-import.pl --server localhost --errata errata.latest.xml --rhsa-oval com.redhat.rhsa-all.xml --publish

exit
