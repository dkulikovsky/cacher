#!/bin/bash
lockfile="/var/tmp/indexer-main"

echo "Starting indexer on path on $(date)"

# lock indexer 
lockfile-create $lockfile 
lockfile-touch $lockfile &
LOCKPID="$!"

# just run indexer
/usr/bin/indexer --rotate path
 
kill "${LOCKPID}"
lockfile-remove $lockfile

echo "Finished indexer on main on $(date)"
