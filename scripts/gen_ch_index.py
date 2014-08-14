#!/usr/bin/env python
import requests, re
from pyhashxx import hashxx

# global
conf = "/etc/cacher/config.ini"

def get_storages(conf):
    f = open(conf,'r')
    stre = re.compile("^\s*storage\s*=\s*(.*)\s*$")
    storages = []
    for line in f.readlines():
        m = stre.match(line)
        if m:
            for st in m.group(1).split(","):
                st = st.strip()
                storages.append(st.split(":")[0])

    if not storages:
        print "Failed to parse storages from %s" % conf
        sys.exit(1)

    return storages


if __name__ == "__main__":
    # print xml
    xml = """<?xml version="1.0" encoding="utf-8"?>
    <sphinx:docset>
    <sphinx:schema>
    <sphinx:field name="path"/>
    <sphinx:attr name="metric" type="string"/>
    <sphinx:attr name="storage" type="string"/>
    </sphinx:schema>\n"""
    print xml
    
    for storage in get_storages(conf):
        step = 1000000
        pos = 0
        while 1:
            # fetch data from clickhouse with limit
            url = "http://%s:8123" % storage
            query = "SELECT DISTINCT(Path) FROM graphite ORDER BY Path LIMIT %d, %d" % (pos, step - 1)
            data = requests.get(url, params = {'query': query})
            if len(data.text) == 0:
                break
            for item in data.text.split("\n"):
                item = str(item)
                if not item:
                    continue
                xml = "<sphinx:document id=\"%d\"><path>%s</path><metric>%s</metric><storage>%s</storage></sphinx:document>" % \
                                     (hashxx(item+storage), item, item, storage)
                print "%s" % xml
        
            pos += step
        
        
    print "</sphinx:docset>"
