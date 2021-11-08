yum groupinstall "Development Tools"
yum install epel-release
yum install openssl11 openssl11-devel
yum install pam-devel
yum install libsq3-devel
yum install wget
wget https://dovecot.org/releases/2.3/dovecot-2.3.17.tar.gz
tar xzvf dovecot-2.3.17.tar.gz
cd dovecot-2.3.17
CPPFLAGS="-I/usr/include/openssl11/" LDFLAGS="-L/usr/lib64/openssl11" ./configure --prefix=/usr --sysconfdir=/etc --with-ssl=openssl --with-sqlite
make
touch /root/checkpoint
make install
find /etc /usr -newer /root/checkpoint -xdev
