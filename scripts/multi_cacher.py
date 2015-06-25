#!/usr/bin/env python
import sys, re, subprocess,string
from os.path import isdir,isfile
from os import mkdir
from time import sleep

# conf files
multiconf = "/etc/cacher/multi.ini"
conf = "/etc/cacher/config.ini"

if not isfile(multiconf):
    print "file %s not found" % multiconf
    sys.exit(1)

if not isfile(conf):
    print "file %s not found" % conf
    sys.exit(1)

# load multiconf and check if config exists
try:
	mc = open(multiconf, "r")
except Exception, e:
	print "Failed to open multiconf %s, error: %s" % (multiconf, e)
	sys.exit(1)

# read config file
conf = {}
opt_re = re.compile("^\s*(\w+)\s*=\s*(\S+)\s*$")
for line in mc.readlines():
	m = opt_re.match(line)
	if m:
		conf[string.lower(m.group(1))] = m.group(2)

for item in ["port", "instances"]:
	if not conf.has_key(item):
		print "%s is not defined in multiconf" % item
		sys.exit(1)
# set defaults
conf['port'] = int(conf['port'])
if not conf.has_key("metricsearch"):
    conf["metricsearch"] = "127.0.0.1"

# check if log dir exists
if not isdir("/var/log/cacher"):
    mkdir("/var/log/cacher")

# start instances
pids = []
for i in xrange(0, conf["instances"]):
    print "Starting %d instance" % i
    pid = subprocess.Popen(["/usr/bin/cacher", "-config", "/etc/cacher/config.ini", "-log",\
                 "/var/log/cacher/cacher_log_%d.log" % (conf["port"]+i),\
                 "-port",  str(conf["port"]+i),\
                 "-metricSearch",  conf["metricsearch"]],\
                 stderr = sys.stderr).pid
    pids.append(pid)

sleep(1)
print "Started %d instances" % len(pids)
print "pids: %s" % pids

