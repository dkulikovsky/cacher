package delta

import (
	"bufio"
	"cacher/mylib"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
    "hash/crc32"
)

type CacheItem struct {
	metric  string
	storage string
}

func deltaSender(metricSearch string, logger *log.Logger, delta chan CacheItem) {
	for {
		item := <-delta
		transport := http.Transport{
			Dial: mylib.DialTimeout,
		}
		client := http.Client{
			Transport: &transport,
		}

        for retry := 0; retry < 3; retry++ {
		    resp, err := client.Get("http://"+metricSearch+":7000/add?name=" + item.metric)
			if err != nil {
				logger.Printf("Error: failed to add metric %s, retries left %d,  err [ %v ]", item, retry,  err)
                continue
			}
            if resp.StatusCode != 200 {
                logger.Printf("Error: failed to add metric %s, retries left %d, err [ %v ]", item, retry,  err)
                continue
            }
            logger.Printf("Debug: added metric %s", item)
			resp.Body.Close()
            break
        }
    }
}

// it is the same function as in mylib, but deadline is set to 10min
func DialTimeoutLong(network, addr string) (net.Conn, error) {
	c, err := net.DialTimeout(network, addr, time.Duration(60*time.Second))
	if err != nil {
		return c, err
	}
	c.SetDeadline(time.Now().Add(time.Duration(600 * time.Second)))
	return c, err
}

func loadCache(metricSearch string, senders []mylib.Sender, logger *log.Logger, cache map[uint32]int) {
	// load cache from file
	resp, err := http.Get("http://"+metricSearch+":7000/dump")
	if err != nil {
		logger.Printf("Error: failed to get index from metricsearch, err [ %v ]", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		m := strings.TrimSpace(scanner.Text())
        if m == "" {
                continue
        }
        mhash := crc32.ChecksumIEEE([]byte(m))
		cache[mhash] = 1
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("something went wrong while scanning through the index, err %v", err)
	}
	return
}

func DeltaManager(metrics chan string, senders []mylib.Sender, metricSearch string, boss mylib.Boss, logger *log.Logger) {
	delta := make(chan CacheItem, 100000)
	cache := make(map[uint32]int)
	go deltaSender(metricSearch, logger, delta)
	logger.Println("loading cache")
	loadCache(metricSearch, senders, logger, cache)
	logger.Printf("loaded %d\n", len(cache))
	for {
		m := <-metrics
        mhash := crc32.ChecksumIEEE([]byte(m))
		_, ok := cache[mhash]
		if !ok {
			// every new metric in deltaManager must have a storage
			storage := ""
			// if there is a single storage everything is easy
			if boss.Single == 1 {
				storage = boss.Senders[0].Host
			} else {
				// but sometimes we can use multiply storage, than it's a hashring task to choose the right one
				r, err := boss.Ring.GetN(m, boss.Rf)
				if err != nil {
					logger.Printf("Failed to get caches for metric %s, err %v\n", m, err)
					continue
				}
				if len(r) > 0 {
					// daleta manager doesn't inquire about replication and awlays will show the first
					// storage in the ring, dublication because of replication is handled by main sphinx index - path
					storage = r[0]
				} else {
					logger.Println("Failed to get storage for some reason :(")
				}
			} // single storage vs multistorage
			delta <- CacheItem{m, storage}
			cache[mhash] = 1
		} // if !ok
	}
}

func BogusDelta(input chan string) {
	for {
		<-input
	}
}
