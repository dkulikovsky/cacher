#!/bin/bash
lockfile="/var/tmp/indexer-delta"

echo "Starting indexer on delta on $(date)"

# lock indexer 
lockfile-create $lockfile 
lockfile-touch $lockfile &
LOCKPID="$!"

# just run indexer
/usr/bin/indexer --noprogress --rotate --merge update delta
/usr/bin/indexer --noprogress --rotate delta
 
kill "${LOCKPID}"
lockfile-remove $lockfile

echo "Finished indexer on main on $(date)"
