package delta

import (
	"bufio"
	"cacher/mylib"
	"log"
	"net/http"
	"net"
	"time"
    "strings"
    "os"
)

type CacheItem struct {
    metric string
    storage string
}

func deltaSender(logger *log.Logger, delta chan CacheItem) {
	for {
            item := <- delta
            resp, err := http.Get("http://127.0.0.1:7000/add?name="+item.metric)
            defer resp.Body.Close()
            if err != nil {
                    logger.Printf("Error: failed to add metric %s, err [ %v ]", item, err)
            } else {
                    logger.Printf("DEBUG: added %s", item)
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

func loadCache(senders []mylib.Sender, logger *log.Logger, cache map[string]int) {
    // load cache from file
    file, err := os.Open("/var/tmp/metrics.dat")
    defer file.Close()
    if err != nil {
        logger.Printf("failed to open metrics file, no cache loaded, err = %v", err)
        return 
    }
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        m := strings.TrimSpace(scanner.Text())
        cache[m] = 1
    }
    if err := scanner.Err(); err != nil {
        logger.Printf("something went wrong while scanning through the index file, err %v", err)
    }
    return
}

func DeltaManager(metrics chan string, senders []mylib.Sender, deltaPort string, boss mylib.Boss, logger *log.Logger) {
    delta := make(chan CacheItem, 100000)
    cache := make(map[string]int)
	go deltaSender(logger, delta)
	logger.Println("loading cache")
	loadCache(senders, logger, cache)
	logger.Printf("loaded %d\n", len(cache))
	for {
		m := <-metrics
		_, ok := cache[m]
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
			cache[m] = 1
		} // if !ok
	}
}

func BogusDelta(input chan string) {
	for {
		<-input
	}
}
