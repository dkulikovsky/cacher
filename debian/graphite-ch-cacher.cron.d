# main index locks main lock
*/5 * * * * root /usr/local/bin/index_main.sh >> /var/log/graphite-ch-cacher/index-main.log 2>>/var/log/graphite-ch-cacher/index-main.err 
# update index locks main and update locks
#1 5 * * * root /usr/local/bin/index_update.sh >> /var/log/graphite-ch-cacher/index-update.log 2>>/var/log/graphite-ch-cacher/index-update.err 
# delta index locks update lock
#*/3 * * * * root /usr/local/bin/index_delta.sh >> /var/log/graphite-ch-cacher/index-delta.log 2>>/var/log/graphite-ch-cacher/index-delta.err 
