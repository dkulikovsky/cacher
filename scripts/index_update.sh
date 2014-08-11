#!/bin/bash
delta_lockfile="/var/tmp/indexer-delta"
main_lockfile="/var/tmp/indexer-main"

echo "Starting indexer on update and delta on $(date)"

# lock main
lockfile-create $main_lockfile 
lockfile-touch $main_lockfile &
MAIN_LOCKPID="$!"

# lock main
lockfile-create $delta_lockfile 
lockfile-touch $delta_lockfile &
DELTA_LOCKPID="$!"

# just run indexer
/usr/bin/indexer --rotate update
/usr/bin/indexer --rotate delta
 
kill "${DELTA_LOCKPID}"
lockfile-remove $delta_lockfile
kill "${MAIN_LOCKPID}"
lockfile-remove $main_lockfile

echo "Finished indexer on update and delta on $(date)"
