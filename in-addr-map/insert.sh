#!/bin/bash

# Pipe this to `sqlite3 in-addr.sql` to create an empty database

cat <<EOF
CREATE TABLE t1 (ipint integer primary key autoincrement, o1 int, o2 int, o3 int, o4 int, rcode int, ptr text, lastupd integer default 0);
CREATE INDEX ptr_index on t1(ptr);
CREATE INDEX lastupd_index on t1(lastupd);
CREATE INDEX nextip on t1(o1, o2, o3, o4, lastupd);
EOF

A=${1:-1}

for B in `seq 0 255`
do
	for C in `seq 0 255`
	do
		echo BEGIN TRANSACTION\;
		for D in `seq 0 255`
		do
			echo INSERT INTO t$A \(o1,o2,o3,o4\) VALUES \($A, $B, $C, $D\)\;
		done
		echo COMMIT\;
	done
	gecho -n . >&2
done
