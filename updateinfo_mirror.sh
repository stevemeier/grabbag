#!/bin/bash

# Change these settings
USERNAME=""
PASSWORD=""
# No changes after this point

BASEURL="https://updateinfo.cefs.steve-meier.de/"
DIRS=( "7/updates/x86_64" "8/updates/aarch64" "8/updates/ppc64le" "8/updates/x86_64" )
WGETOPTS="-N -T 30 -q"

# Rewrite %40 to @, if necessary
USERNAME=${USERNAME/'%40'/@}

# Check that username and password are set
if [[ -z "${USERNAME}" || -z "${PASSWORD}" ]]; then
	echo "ERROR: Please sign up via Patreon to receive a username and password"
	exit 2
fi

# Check for that destination is an existing directory
if [ ! -d "${1}" ]; then
	echo "ERROR: Please specify a destination directory"
	exit 1
fi

# Iterate through all supported versions and archs
cd ${1}
for DIR in "${DIRS[@]}"
do
	# Create the repodata/ subdirectory
	if [ ! -d ${DIR}/repodata ]; then
		mkdir -p ${DIR}/repodata
	fi

	# Get the repomd.xml which contains the index
	wget ${WGETOPTS} -P ${1}/${DIR}/repodata ${BASEURL}${DIR}/repodata/repomd.xml

	# Extract the other filenames from repomd.xm and fetch them
	for REPOFILE in $(grep href ${1}/${DIR}/repodata/repomd.xml | cut -d \" -f 2)
	do
		wget ${WGETOPTS} --user ${USERNAME} --password ${PASSWORD} -P ${1}/${DIR}/repodata ${BASEURL}${DIR}/${REPOFILE}
		if [ $? -eq 6 ]; then
			echo "ERROR: Your username/password are incorrect"
			exit 2
		fi
	done
done
