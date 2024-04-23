#!/bin/bash

DBPATH="${1}"
if [ "${DBPATH}" == "" ]; then
	DBPATH="/home/mail/followup.db"
fi

IFS='/' read -ra DIRS <<< "${DBPATH}"
cd /
for DIR in "${DIRS[@]}"; do
	if [ ! -z "${DIR}" ] && [ ! -r "${DIR}" ]; then
		echo "UNKNOWN - Can not access ${DIR} en route to ${DBPATH}"
		exit 3
	fi
        if [ -d "${DIR}" ]; then
		cd ${DIR}
	fi
done

if [ ! -f ${DBPATH} ]; then
	echo "UNKNOWN - $DBPATH not found"
	exit 3
fi

EPOCH=$(date +%s)
QUEUESIZE=`echo 'SELECT COUNT(*) FROM reminders WHERE status IS null;' | sqlite3 ${DBPATH}`
NOTSEND=`echo 'SELECT MIN(timestamp) FROM reminders WHERE status IS null;' | sqlite3 ${DBPATH}`

if [ "${NOTSEND}" == "" ]; then
	echo "OK - No pending reminders"
	exit 0
fi
if [ "${EPOCH}" -le ${NOTSEND} ]; then
	echo "OK - ${QUEUESIZE} Reminder(s) in queue"
	exit 0
fi
if [ "${EPOCH}" -gt ${NOTSEND} ]; then
	echo "CRITICAL - Reminder not send"
	exit 2
fi
