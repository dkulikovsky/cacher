package main

import (
	"bufio"
	"cacher/delta"
	"cacher/mylib"
	"flag"
	"fmt"
	"github.com/stathat/consistent"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

import _ "net/http/pprof"

const (
	DEBUG  = 1
	layout = "Jan 2, 2006 at 15:04:02"
)

var (
	listen_port string  = "8765"
	confFile    *string = flag.String("config", "config.ini", "config file")
	listenPort  *string = flag.String("port", "8765", "listen port for incoming data")
	metricSearch   *string = flag.String("metricSearch", "9876", "port for delta handler")
	logFile     *string = flag.String("log", "/var/log/cacher.log", "port for delta handler")
	logger      *log.Logger
	con_alive   int64 = 0
)

func process_connection(local net.Conn, boss mylib.Boss, mon *mylib.Mmon) {
	defer local.Close()
	local.SetReadDeadline(time.Now().Add(60 * time.Second))

	scanner := bufio.NewScanner(local)
	for scanner.Scan() {
		atomic.AddInt32(&mon.Rcv, 1)
		parse(scanner.Text(), boss)

		local.SetReadDeadline(time.Now().Add(60 * time.Second))
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("socket error: %v", err)
	}

	atomic.AddInt64(&con_alive, -1)
}

func metricTest(s string) func(rune) bool {
	return func(r rune) bool {
		return (!strings.ContainsRune(s, r) &&
			r != '.' &&
			r != '_' && r != '-' &&
			(r < '0' || r > '9') &&
			(r < 'A' || r > 'Z') &&
			(r < 'a' || r > 'z'));
	}
}

func parse(input string, boss mylib.Boss) {
	if len(input) < 15 {
		// input str must be at least 15 chars
		return
	}
	input = strings.Trim(input, " ")

	var metric, data, ts string
	//format: metric.path value timestamp
	arr := strings.Fields(input)
	if len(arr) == 3 {
		metric, data, ts = arr[0], arr[1], arr[2]
		// validate metric name
		if strings.IndexFunc(metric, boss.MetricFunc) != -1 {
			logger.Printf("Failed to parse metric %s", metric)
		}
		// convert timestamp to int64
		ts_int, err := strconv.ParseInt(ts, 0, 64)
		if err != nil {
			logger.Println("Failed to parse to int")
			return
		}
		t := time.Unix(ts_int, 0)
		curr_time := time.Now()

		// just check if data is a real float number
		// but pass to clickhouse as a string to avoid useless convertation
		_, err = strconv.ParseFloat(data, 32)
		if err != nil {
			logger.Printf("Failed to parse data to float, bogus string [ %s ]", input)
			return
		}

		if boss.Single == 1 {
			w := singleSender(boss.Senders)
			w.Pipe <- fmt.Sprintf("%s\t%s\t%d\t%s\t%d\n", metric, data, t.Unix(), t.Format("2006-01-02"), curr_time.Unix())
		} else {
			r, err := boss.Ring.GetN(metric, boss.Rf)
			if err != nil {
				logger.Printf("Failed to get caches for metric %s, err %v\n", metric, err)
				return
			}

			for _, item := range r {
				w := getSender(item, boss.Senders)
				w.Pipe <- fmt.Sprintf("%s\t%s\t%d\t%s\t%d\n", metric, data, t.Unix(), t.Format("2006-01-02"), curr_time.Unix())
			}
		}
		// send metric to deltaManager
		boss.DeltaChan <- metric
	} else {
		logger.Printf("[Error] Bad formated input: %s\n", input)
	}
	return
}

func sender(sender mylib.Sender, send_mon *int32) {
	logger.Printf("Started sender with options: [%s:%d instance %d]", sender.Host, sender.Port, sender.Index)
	data_buf := make([]string, 0, 500000)
	var send int32
	send = 0
	ticker := time.Tick(5 * time.Second)
	for {
		select {
		case <-ticker:
			if len(data_buf) > 0 {
				//	log("debug", fmt.Sprintf("sending (t) %d bytes to %s..", data_buf.Len(), sender.host))
				send_data(strings.Join(data_buf, ""), sender)
				atomic.AddInt32(send_mon, send)
				send = 0
				// reset buffer
				data_buf = nil
				data_buf = make([]string, 0, 500000)
			}
		case input_buf := <-sender.Pipe:
			data_buf = append(data_buf, input_buf)
			send++
		}
	}
}

func send_data(data string, c mylib.Sender) {
	// an ugly hack to handle conn and write timeouts
	transport := http.Transport{
		Dial: mylib.DialTimeout,
	}
	client := http.Client{
		Transport: &transport,
	}

	url := fmt.Sprintf("http://%s:%d", c.Host, c.Port)

	for retry := 0; retry < 3; retry++ {
		req := strings.NewReader(fmt.Sprintf("INSERT INTO default.graphite FORMAT TabSeparated\n%s", data))
		resp, err := client.Post(url, "text/xml", req)
		if err != nil {
			logger.Printf("Failed to send data to %s:%d, retries left %d, going to sleep for 1s, error: %v", c.Host, c.Port, retry, err)
			time.Sleep(1000*time.Millisecond)
			continue
		}
		_, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.Printf("Failed to read response body, retries left %d, going to sleep for 1s, error: %v", retry, err)
			time.Sleep(1000*time.Millisecond)
			continue
		}
		if resp.StatusCode != 200 {
			logger.Printf("Got not 200 response status for %s:%d, retries left %d, going to sleep for 1s, status: %s", c.Host, c.Port, retry, resp.Status)
			time.Sleep(1000*time.Millisecond)
			continue
		}
		break
	}
	//	status := resp.StatusCode
	//    log("debug", fmt.Sprintf("executer insert, status %d, host %s:%d:%d, len %d", status, c.host,c.port,c.index, len(data)))
}

func singleSender(senders []mylib.Sender) mylib.Sender {
	// if we just have multiply senders for one host, than senders will be
	// randomly rotated
	return senders[rand.Intn(len(senders))]
}

func getSender(r string, senders []mylib.Sender) mylib.Sender {
	var senderArr []mylib.Sender
	for _, w := range senders {
		if r == w.Host {
			senderArr = append(senderArr, w)
		}
	}
	randIndex := rand.Intn(len(senderArr))

	return senderArr[randIndex]
}

// functions for monitoring internals
func monitor(mon *mylib.Mmon, boss mylib.Boss) {
	// just pick up first sender all the time, kiss
	sender := boss.Senders[0]
	ticker := time.Tick(1000 * time.Millisecond)
	last := time.Now()
	for {
		select {
		case <-ticker:
			//			log("debug", fmt.Sprintf("sending to %s..", sender.host))
			curr_time := time.Now()
			if curr_time.Unix() > last.Unix() {
				send_mon_data(atomic.SwapInt32(&mon.Send, 0), atomic.SwapInt32(&mon.Rcv, 0), atomic.SwapInt32(&mon.Conn, 0), boss, curr_time, sender)
				last = curr_time
			}
		}
	}
}

func send_mon_data(m int32, r int32, c int32, boss mylib.Boss, ts time.Time, sender mylib.Sender) {
	// get memstats
	mem := new(runtime.MemStats)
	runtime.ReadMemStats(mem)
	data := map[string]int64{
		"metrics_send": int64(m),
		"metrics_rcvd": int64(r),
		"conn":         int64(c),
		"conn_alive":   atomic.LoadInt64(&con_alive),
		"heap_sys":     int64(mem.HeapSys),
		"heap_idle":    int64(mem.HeapIdle),
		"heap_inuse":   int64(mem.HeapInuse),
		"alloc":        int64(mem.Alloc),
		"sys":          int64(mem.Sys),
	}

	out := ""
	tsF := ts.Format("2006-01-02")
	for key, val := range data {
		metric_name := fmt.Sprintf("one_sec.int_%s.%s", boss.Port, key)
		out += fmt.Sprintf("%s\t%d\t%d\t%s\t%d\n", metric_name, val, ts.Unix(), tsF, ts.Unix())
		// all monitor metrics must exist in metricsearch index
		boss.DeltaChan <- metric_name
	}
	//logger.Printf("DEBUG: monitoring output %s", out)
	send_data(out, sender)
}

func startLogger(logf string) *log.Logger {
	// start logger
	fmt.Printf("strating logger with %s\n", logf)
	logF, err := os.OpenFile(logf, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0655)
	if err != nil {
		fmt.Printf("Unable to create file %s, %v\n", logf, err)
		os.Exit(1)
	} else {
		fmt.Printf("opened log file %s\n", logf)
	}

	logger = log.New(logF, "cacher: ", log.Ldate|log.Ltime)
	//logger = log.New(os.Stdout, "cacher: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Print("Starting")
	return logger
}

func startWorkers(config mylib.Config, r *consistent.Consistent, mon *mylib.Mmon) []mylib.Sender {
	workers := make([]mylib.Sender, 0)
	index := 0
	for _, st := range config.Storages {
		for j := 0; j <= st.Num; j++ {
			var w mylib.Sender
			w.Host = st.Host
			w.Port = st.Port
			w.Pipe = make(chan string, config.ChanLimit)
			w.Index = index
			r.Add(st.Host)
			index++
			workers = append(workers, w)
			go sender(w, &mon.Send)
		}
	}
	return workers
}

func freemem() {
	ticker := time.Tick(1000 * time.Millisecond)
	for {
		select {
		case <-ticker:
			logger.Println("just checking")
		}
	}
}

func main() {
	runtime.GOMAXPROCS(4)
	debug.SetGCPercent(90)
	//	go freemem()

	rand.Seed(time.Now().Unix())
	// parse config
	flag.Parse()
	if flag.NFlag() != 4 {
		fmt.Printf("usage: cacher -config config_file -log log_file -port listen_port -metricSearch metric_search_host\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	logger := startLogger(*logFile)

	var config mylib.Config
	if _, err := os.Stat(*confFile); os.IsNotExist(err) {
		logger.Printf("no such file: %s, loading default\n", *confFile)
		config = mylib.Load("")
	} else {
		config = mylib.Load(*confFile)
		logger.Printf("using %s as config file\n", *confFile)
	}

	// set hash ring object
	r := consistent.New()
	// set up monitoring
	mon := new(mylib.Mmon)
	// spawn db writers and fill hash ring object
	workers := startWorkers(config, r, mon)

	var boss mylib.Boss
	deltaChan := make(chan string, 5000000)
	// create Boss var (used to hide tons of vars in functions stack)
	boss.Senders = workers
	boss.Rf = config.Rf
	boss.Ring = r
	boss.Single = 0
	boss.Port = *listenPort
	boss.DeltaChan = deltaChan
	if (config.MetricCheck) {
		boss.MetricFunc = metricTest(config.MetricChars) 
	} else {
		boss.MetricFunc = func(rune) bool {
			return false
		}
	}
	// if we have a single host, than we can ignore hash ring mess
	// and do simple rr rotation of senders
	if len(boss.Ring.Members()) == 1 {
		boss.Single = 1
	}
	// start delta manager
	if config.EnableDelta > 0 {
		go delta.DeltaManager(deltaChan, workers, *metricSearch, boss, logger)
		go func() {
			http.ListenAndServe(":5"+*listenPort, nil)
		}()
	} else {
		go delta.BogusDelta(deltaChan)
	}

	go monitor(mon, boss)

	// start listener
	ln, err := net.Listen("tcp", ":"+*listenPort)
	logger.Printf("Started on %s port\n", *listenPort)
	logger.Printf("worker chanLimit %d\n", config.ChanLimit)
	if err != nil {
		logger.Fatalf("Unable to start listener, %v\n", err)
	}

	// main loop
	for {
		conn, err := ln.Accept()
		if err == nil {
			go process_connection(conn, boss, mon)
			// received new connection
			atomic.AddInt32(&mon.Conn, 1)
			atomic.AddInt64(&con_alive, 1)
		} else {
			logger.Printf("Failed to accept connection, %v\n", err)
		}
	}

	logger.Println("Done")
}
