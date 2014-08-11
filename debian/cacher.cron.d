# main index locks main lock
1 2 * * * root /usr/local/bin/index_main.sh > /var/log/cacher/index-main.log 2>/var/log/cacher/index-main.err 
# update index locks main and update locks
1 5 * * * root /usr/local/bin/index_update.sh > /var/log/cacher/index-update.log 2>/var/log/cacher/index-update.err 
# delta index locks update lock
*/3 * * * * root /usr/local/bin/index_delta.sh > /var/log/cacher/index-delta.log 2>/var/log/cacher/index-delta.err 
