#!/usr/bin/env python
import ceres
import os
import sys
import time
import re
import requests
from os.path import basename, dirname, splitext, exists, join, isfile, expanduser, abspath
from optparse import OptionParser

def log_msg(lvl, msg):
    print "%s [%s] %s\n" % (time.ctime(), lvl, msg)

def read_data(sfile):
    path = "/".join(sfile.split("/")[:-1])
    # get files options from path
    # path = /var/tmp/node_reader/errors_relative/1408110900@60.slice
    # slice_file = 1408110900@60.slice, stime = 1408110900, step = 60
    slice_file = sfile.split('/')[-1]
    stime = slice_file.split('@')[0]
    step = slice_file.split('@')[1].split('.')[0]
    # read data
    class NodeTmp():
        def __init__(self, path):
            self.fsPath = path
    s = ceres.CeresSlice(NodeTmp(path), int(stime), int(step))
    data = s.read(int(stime), s.endTime)
    # we got timeSeries object as result, now we need to convert it to dict with timestamps
    result = {}
    i = 0
    for ts in xrange(data.startTime, data.endTime, data.timeStep):
        if not data.values[i]:
            data.values[i] = 0
        result[ts] = data.values[i]
        i += 1
    return result

def get_oldest_ts(storage, m):
    try:
        res = requests.post("http://%s" % storage, "SELECT min(Time) FROM default.graphite WHERE Path == '%s'" % m)
    except Exception, e:
        print "Failed to send data to CH, err: %s" % e
        return 0
    if res.status_code == 200 and res.text:
        return int(res.text)
    else:
        return 0

def to_date(ts):
    return time.strftime("%Y-%m-%d", time.localtime(ts))

def send_data(storage, metric, data):
    ts_arr = sorted(data.keys())
    if not ts_arr:
        return
    # get oldest timestamp from clickhouse
    oldest = get_oldest_ts(storage, metric)
    if not oldest:
        oldest = max(ts_arr)
    output = []
    for ts in ts_arr:
        if ts >= oldest:
            break
        output.append("('%s', %s, %s, '%s')" % (metric, data[ts], ts, to_date(ts)))
    query = "INSERT INTO default.graphite VALUES %s" % ",".join(output)
    try:
        res = requests.post("http://%s" % storage, query)
    except Exception, e:
        print "Failed to send data to CH, err: %s" % e
        return 

    if res.status_code == 200:
        print "Succeded to write %d lines to CH for %s" % (len(output), metric)
    else:
        print "Something.. i don't really know what.. %s" % res

if __name__ == '__main__':
    parser = OptionParser()
    parser.add_option("-r","--root",action="store",type="string",dest="root",help="root dir to start traverse from")
    parser.add_option("-d","--destitantion",action="store",type="string",dest="storage",default="127.0.0.1:8123", help="address for clickhouse storage 127.0.0.1:8123")
    parser.add_option("-p","--prefix",action="store",type="string",dest="prefix",help="metrics prefix")
    (options, args) = parser.parse_args()

    if not options.root:
        print "root is required"
        sys.exit(1)
    if not options.prefix:
        print "prefix is required"
        sys.exit(1)

    for folder, subs, files in os.walk(options.root):
        if exists(join(folder, '.ceres-node')):
            log_msg("info", "found node %s, working with slices" % folder)
            node_data = {}
            # it's a node
            for sfile in files:
                if re.match(r'.*\.slice$',sfile):
                    node_data.update(read_data(join(folder, sfile)))
            # prepare metric
            metric = folder[len(options.root)+1:]
            metric = metric.replace("/",".")
            metric = "%s.%s" % (options.prefix, metric)
            send_data(options.storage, metric, node_data)
                
            

