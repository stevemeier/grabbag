#!/bin/sh

if [ ! -x /sbin/rpmconf ]; then
  echo "UNKNOWN: rpmconf binary not found"
  exit 3
fi

NEWCONFS=$(rpmconf -ta 2>/dev/null | wc -l)
if [ "${NEWCONFS}" -eq 0 ]; then
  echo "OK: No rpmnew configs found"
  exit 0
fi

CONFS=$(rpmconf -ta 2>/dev/null | sed s/\.rpmnew$//g)
PENDING=$(rpm --queryformat '%{NAME}\n' -qf $CONFS | sort -u | tr "\n" " ")

echo "WARNING: Pending configs for: $PENDING"
exit 1
