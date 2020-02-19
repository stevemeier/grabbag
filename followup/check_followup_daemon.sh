#!/bin/bash

DBPATH="${1}"
if [ "${DBPATH}" == "" ]; then
	DBPATH="/home/mail/followup.db"
fi

if [ ! -f ${DBPATH} ]; then
	echo "UNKNOWN - $DBPATH not found"
	exit 3
fi
if [ ! -r ${DBPATH} ]; then
	echo "UNKNOWN - $DBPATH not readable"
	exit 3
fi

EPOCH=$(date +%s)
NOTSEND=`echo 'SELECT MIN(timestamp) FROM reminders WHERE status IS null;' | sqlite3 ${DBPATH}`

if [ "${NOTSEND}" == "" ]; then
	echo "OK - No pending reminders"
	exit 0
fi
if [ "${EPOCH}" -le ${NOTSEND} ]; then
	echo "OK - Reminder in queue"
	exit 0
fi
if [ "${EPOCH}" -gt ${NOTSEND} ]; then
	echo "CRITICAL - Reminder not send"
	exit 2
fi
