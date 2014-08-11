#!/usr/bin/env python
import requests
from pyhashxx import hashxx

xml = """<?xml version="1.0" encoding="utf-8"?>
<sphinx:docset>
<sphinx:schema>
<sphinx:field name="path"/>
<sphinx:attr name="metric" type="string"/>
</sphinx:schema>\n"""
print xml

step = 100000
pos = 0
while 1:
    # fetch data from clickhouse with limit
    data = requests.get("http://127.0.0.1:8123", params = {'query': "SELECT DISTINCT(Path) FROM graphite LIMIT %d, %d" % (pos, step - 1)})
    if len(data.text) == 0:
        break
    for item in data.text.split("\n"):
        item = str(item)
        if not item:
            continue
        xml = "<sphinx:document id=\"%d\"><path>%s</path><metric>%s</metric></sphinx:document>" % (hashxx(item), item, item)
        print "%s" % xml

    pos += step


print "</sphinx:docset>"
