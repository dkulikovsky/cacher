package delta

import (
	"bufio"
	"bytes"
	"cacher/mylib"
	"fmt"
	"log"
	"net/http"
	"net"
	"time"
)

type CacheItem struct {
    host string
    storage string
}

const MAX_DELTA_SIZE = 100000 // limit on delta size

func deltaHandler(w http.ResponseWriter, r *http.Request, logger *log.Logger, delta chan CacheItem) {
	result := ""
    L:
	for {
        select {
            case item := <- delta:
                result += fmt.Sprintf("%s %s\n", item.host, item.storage)
            default:
                break L
        }
	}
	fmt.Fprintf(w, result)
}

func deltaServer(port string, logger *log.Logger, delta chan CacheItem) {
	http.HandleFunc("/delta", func(w http.ResponseWriter, r *http.Request) {
            deltaHandler(w, r, logger, delta)
    })
	logger.Printf("Starting delta server on %s port\n", port)
	err := http.ListenAndServe("0.0.0.0:"+port, nil)
	if err != nil {
		logger.Printf("failed to start delta server, err = %v\n", err)
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
	// ugly way to set timeout
	transport := http.Transport{
		Dial: DialTimeoutLong,
	}
	client := http.Client{
		Transport: &transport,
	}

	hosts_done := make(map[string]int)
	for _, w := range senders {
		_, ok := hosts_done[w.Host]
		if ok {
			continue
		} else {
			hosts_done[w.Host] = 1
		}

		url := fmt.Sprintf("http://%s:%d", w.Host, w.Port)
		req := fmt.Sprintf("SELECT Distinct(Path) from graphite")
		resp, err := client.Post(url, "text/xml", bytes.NewBufferString(req))
		defer resp.Body.Close()

		if err != nil {
			logger.Printf("failed to load cache from %s, error: %v\n", w.Host, err)
			continue
		}
		body := bufio.NewScanner(resp.Body)
		for body.Scan() {
			line := body.Text()
			cache[line] = 1
		}
		if err := body.Err(); err != nil {
			logger.Printf("failed to parse response from %s:%d, err: %v\n", w.Host, w.Port, err)
		}
		logger.Printf("loaded data from %s:%d, cache size now %d\n", w.Host, w.Port, len(cache))
	}
	return 
}

func DeltaManager(metrics chan string, senders []mylib.Sender, deltaPort string, boss mylib.Boss, logger *log.Logger) {
    delta := make(chan CacheItem, 100000)
    cache := make(map[string]int)
	go deltaServer(deltaPort, logger, delta)
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
