#!/usr/bin/env python
import requests
from pyhashxx import hashxx
import sys, re, subprocess,string
from os.path import isdir,isfile
from os import mkdir
from time import sleep

# conf files
multiconf = "/etc/cacher/multi.conf"

if not isfile(multiconf):
    print "file %s not found" % multiconf
    sys.exit(1)

# load multiconf and check if config exists
try:
	mc = open(multiconf, "r")
except Exception, e:
	print "Failed to open multiconf %s, error: %s" % (multiconf, e)
	sys.exit(1)

# read config file
conf = {}
opt_re = re.compile("^\s*(\w+)\s*=\s*(\d+)\s*$")
for line in mc.readlines():
	m = opt_re.match(line)
	if m:
		conf[string.lower(m.group(1))] = int(m.group(2))

for item in ["port", "deltaport", "instances"]:
	if not conf.has_key(item):
		print "%s is not defined in multiconf" % item
		sys.exit(1)

xml = """<?xml version="1.0" encoding="utf-8"?>
<sphinx:docset>
<sphinx:schema>
<sphinx:field name="path"/>
<sphinx:attr name="metric" type="string"/>
</sphinx:schema>\n"""
print xml

step = 100000
pos = 0
for port in xrange(conf["deltaport"], conf["deltaport"] + conf["instances"]):
    # fetch data from clickhouse with limit
    data = requests.get("http://127.0.0.1:%d/delta" % port)
    if len(data.text) == 0:
        continue 
    for item in data.text.split("\n"):
        item = str(item)
        if not item:
            continue
        xml = "<sphinx:document id=\"%d\"><path>%s</path><metric>%s</metric></sphinx:document>" % (hashxx(item), item, item)
        print "%s" % xml

    pos += step


print "</sphinx:docset>"
