#!/bin/bash

# Sync channels?
SYNC=0

# Temporary directory for downloads
TEMPDIR="/tmp"

# curl options
CURLOPTS="-s -m 5"

# Identify CentOS Version
CENTOSVERSION=$(rpm -q centos-release --qf '%{VERSION}' 2>/dev/null)

# Repository URL
REPOURL="http://yum.spacewalkproject.org"

# Check minimum CentOS Version
if [ ${CENTOSVERSION} -lt 6 ]; then
  echo "ERROR: Sorry, CentOS Version ${CENTOSVERSION} is not supported"
  exit 1
fi

## Import Spacewalk GPG KEY
#export KEYYEAR=2015
#if [ ! -f /etc/pki/rpm-gpg/RPM-GPG-KEY-spacewalk-${KEYYEAR} ]; then 
#  echo
#  echo "##################################"
#  echo "## Installing Spacewalk GPG Key ##"
#  echo "##################################"
#  (cd /etc/pki/rpm-gpg && curl ${CURLOPTS} -O ${REPOURL}/RPM-GPG-KEY-spacewalk-${KEYYEAR})
#  rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-spacewalk-${KEYYEAR}
#fi


# Install Spacewalk repo
if [ ! -f /etc/yum.repos.d/spacewalk.repo ]; then
  echo
  echo "#####################################"
  echo "## Installing Spacewalk Repository ##"
  echo "#####################################"
  (cd ${TEMPDIR} && curl ${CURLOPTS} -O https://copr-be.cloud.fedoraproject.org/results/%40spacewalkproject/spacewalk-2.8/epel-${CENTOSVERSION}-x86_64/00736372-spacewalk-repo/spacewalk-repo-2.8-11.el${CENTOSVERSION}.centos.noarch.rpm)
  rpm -Uvh ${TEMPDIR}/spacewalk-repo-*.noarch.rpm || exit 1
  rm -f ${TEMPDIR}/spacewalk-repo-*.noarch.rpm
fi

# Install EPEL (for dependencies)
if [ ! -f /etc/yum.repos.d/epel.repo ]; then
  echo
  echo "################################"
  echo "## Installing EPEL Repository ##"
  echo "################################"
  (cd ${TEMPDIR} && curl ${CURLOPTS} -O https://dl.fedoraproject.org/pub/epel/epel-release-latest-${CENTOSVERSION}.noarch.rpm)
  rpm -Uvh ${TEMPDIR}/epel-release-*.noarch.rpm || exit 1
  rm -f ${TEMPDIR}/epel-release-*.noarch.rpm 
fi

# Spacewalk Client Tools (for dependencies, too)
if [ ! -f /etc/yum.repos.d/spacewalk-client.repo ]; then
  echo
  echo "############################################"
  echo "## Installing Spacewalk-Client Repository ##"
  echo "############################################"
  (cd ${TEMPDIR} && curl ${CURLOPTS} -O https://copr-be.cloud.fedoraproject.org/results/%40spacewalkproject/spacewalk-2.8/epel-${CENTOSVERSION}-x86_64/00736372-spacewalk-repo/spacewalk-client-repo-2.8-11.el${CENTOSVERSION}.centos.noarch.rpm)
  rpm -ihv ${TEMPDIR}/spacewalk-client-repo-*.noarch.rpm || exit 1
  rm -f ${TEMPDIR}/spacewalk-client-repo-*.noarch.rpm
fi

## Add Java repository
#cd /etc/yum.repos.d
#if [ "${CENTOSVERSION}" -eq 6 ]; then
#  curl ${CURLOPTS} -O https://copr.fedorainfracloud.org/coprs/g/spacewalkproject/java-packages/repo/epel-7/group_spacewalkproject-java-packages-epel-7.repo
#  curl ${CURLOPTS} -O https://copr.fedorainfracloud.org/coprs/g/spacewalkproject/epel6-addons/repo/epel-6/group_spacewalkproject-epel6-addons-epel-6.repo
#fi
#if [ "${CENTOSVERSION}" -eq 7 ]; then
#  curl ${CURLOPTS} -O https://copr.fedorainfracloud.org/coprs/g/spacewalkproject/java-packages/repo/epel-7/group_spacewalkproject-java-packages-epel-7.repo
#fi

## Import RED HAT GPG Key
#if [ ! -f /etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release ]; then 
#  echo
#  echo "################################"
#  echo "## Installing Red Hat GPG Key ##"
#  echo "################################"
#  (cd /etc/pki/rpm-gpg && curl ${CURLOPTS} https://www.redhat.com/security/data/37017186.txt > /etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release)
#  rpm --import /etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
#fi

# Install Spacewalk
rpm -qi spacewalk-common 2>&1 >/dev/null
if [ $? -eq 1 ]; then
  echo
  echo "##########################"
  echo "## Installing Spacewalk ##"
  echo "##########################"
  yum --nogpgcheck -y install spacewalk-postgresql spacewalk-setup-postgresql spacewalk-utils spacecmd
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
    firewall-cmd -q --permanent --add-service=http
    firewall-cmd -q --permanent --add-service=https
    firewall-cmd -q --reload
  fi
fi

# Setup database first to work around this bug:
# https://bugzilla.redhat.com/show_bug.cgi?id=1524221
if [ -d /var/lib/pgsql ]; then
  echo
  echo "###########################"
  echo "## Initializing database ##"
  echo "###########################"
  rm -rf /var/lib/pgsql/data
  /usr/bin/postgresql-setup initdb
fi

# Run setup
grep db_password /etc/rhn/rhn.conf > /dev/null
if [ $? -eq 1 ]; then
  echo
  echo "##########################"
  echo "## Setting up Spacewalk ##"
  echo "##########################"
  /usr/bin/spacewalk-setup --answer-file=${ANSWERFILE}
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
if [ ! -f CentOS-6.9-x86_64-netinstall.iso ]; then
  curl ${CURLOPTS} -O http://mirror.rackspace.com/CentOS/6/isos/x86_64/CentOS-6.9-x86_64-netinstall.iso
fi
if [ ! -f CentOS-7-x86_64-NetInstall-1708.iso ]; then
  curl ${CURLOPTS} -O http://mirror.rackspace.com/CentOS/7/isos/x86_64/CentOS-7-x86_64-NetInstall-1708.iso
fi

grep CentOS-6.9-x86_64-netinstall.iso /etc/fstab > /dev/null
if [ $? -ne 0 ]; then
cat >> /etc/fstab <<EOF
/var/iso-images/CentOS-6.9-x86_64-netinstall.iso           /var/distro-trees/CentOS-6-x86_64       iso9660 loop,ro 0 0
EOF
fi

grep CentOS-7-x86_64-NetInstall-1708.iso /etc/fstab > /dev/null
if [ $? -ne 0 ]; then
cat >> /etc/fstab <<EOF
/var/iso-images/CentOS-7-x86_64-NetInstall-1708.iso        /var/distro-trees/CentOS-7-x86_64       iso9660 loop,ro 0 0
EOF
fi

echo
echo "##########################"
echo "## Mounting CentOS ISOs ##"
echo "##########################"
grep /var/distro-trees/CentOS-6-x86_64 /etc/mtab > /dev/null
if [ $? -ne 0 ]; then
  mount /var/distro-trees/CentOS-6-x86_64
fi

grep /var/distro-trees/CentOS-7-x86_64 /etc/mtab > /dev/null
if [ $? -ne 0 ]; then
  mount /var/distro-trees/CentOS-7-x86_64
fi

echo
echo "============================="
echo "=== INSTALLATION COMPLETE ==="
echo "============================="
echo

echo
echo "###########################"
echo "## Setting up admin user ##"
echo "###########################"
echo
SWUSER=admin
SWPASS=admin1
TEMPFILE=$(mktemp)

# from https://gist.github.com/vinzent/4bba600573bc9eeb33c4#gistcomment-1810454
if [ "$(satwho | wc -l)" = "0" ]; then
  curl --silent https://localhost/rhn/newlogin/CreateFirstUser.do --insecure -D - >${TEMPFILE}

  cookie=$(egrep -o 'JSESSIONID=[^ ]+' ${TEMPFILE})
  csrf=$(egrep csrf_token ${TEMPFILE} | egrep -o 'value=[^ ]+' | egrep -o '[0-9]+')

    curl --noproxy '*' \
      --cookie "$cookie" \
      --insecure \
      --data "csrf_token=-${csrf}&submitted=true&orgName=DefaultOrganization&login=${SWUSER}&desiredpassword=${SWPASS}&desiredpasswordConfirm=${SWPASS}&email=root%40localhost&prefix=Mr.&firstNames=Administrator&lastName=Spacewalk&" \
      https://localhost/rhn/newlogin/CreateFirstUser.do

  if [ "$(satwho | wc -l)" = "0" ]; then
    echo "Error: user creation failed" >&2
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
  # spacewalk-repo-sync -e 'xorg*,libreoffice*,thunderbird*,firefox*,autocorr*,java*,kernel-debug*' --channel centos6-i386-updates
  # spacewalk-repo-sync --channel centos6-x86_64-updates
  # spacewalk-repo-sync --channel centos7-x86_64-updates
  # spacewalk-repo-sync --channel centos6-i386
  # spacewalk-repo-sync --channel centos6-x86_64
  # spacewalk-repo-sync --channel centos7-x86_64
fi

echo 
echo "################################"
echo "## Installing useful packages ##"
echo "################################"
echo
yum -y install wget perl-Frontier-RPC perl-Text-Unidecode 
cd
wget http://cefs.steve-meier.de/errata.latest.xml
wget http://cefs.steve-meier.de/errata-import.tar
tar xf errata-import.tar

exit
