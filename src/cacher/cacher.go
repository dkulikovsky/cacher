package main

import (
	"bufio"
	"bytes"
	"cacher/mylib"
	"fmt"
	"github.com/stathat/consistent"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
    "flag"
//	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

import _ "net/http/pprof"

const (
	DEBUG            = 2
	log_name         = "./cacher.log"
    layout           = "Jan 2, 2006 at 15:04:02"
	worker_chan_size = 500000
)

var (
	listen_port string = "8765"
	LogC        chan *Msg
    confFile        *string = flag.String("config", "config.ini", "config file")
)

type Sender struct {
	port  int
	host  string
	pipe  chan string
	index int
}

type Boss struct {
	senders []Sender
	rf      int
	ring    *consistent.Consistent
	single  int
    port string
}

type Mmon struct {
	send int32
	rcv  int32
	conn int32
}

type Msg struct {
	lvl string
	msg string
}

func log(lvl string, msg string) {
	if DEBUG == 1 {
		LogC <- &Msg{lvl, msg}
		return
	}
	if DEBUG == 2 {
		fmt.Printf("%s [%s] %s\n", time.Now().Format(layout), strings.ToLower(lvl), strings.ToLower(msg))
		return
	}
	if strings.ToLower(lvl) != "debug" {
		LogC <- &Msg{lvl, msg}
	}
}

func logger_loop(data chan *Msg) {
	f, err := os.Create(log_name)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Unable to create file %s, %v\n", log_name, err))
		os.Exit(1)
	} else {
		fmt.Sprintf("opened log file %s\n", log_name)
	}
	defer f.Close()
	for {
		b := <-data
		if len(b.msg) == 0 {
			break
		}
		out := fmt.Sprintf("%s [%s] %s\n", time.Now().Format(layout), strings.ToLower(b.lvl), strings.ToLower(b.msg))
		f.Write([]byte(out))
		f.Sync()
	}
}

func printable_addr(a net.Addr) string {
	return strings.Replace(a.String(), ":", "-", -1)
}

func format_time(t time.Time) string {
	return t.Format("2006.01.02-15.04.05")
}

func dialTimeout(network, addr string) (net.Conn, error) {
	c, err := net.DialTimeout(network, addr, time.Duration(30*time.Second))
	if err != nil {
		return c, err
	}
	c.SetDeadline(time.Now().Add(time.Duration(30 * time.Second)))
	return c, err
}

func send_data(data string, c Sender) {
	req := bytes.NewBufferString(fmt.Sprintf("INSERT INTO default.graphite VALUES %s", data))
	// an ugly hack to handle conn and write timeouts
	transport := http.Transport{
		Dial: dialTimeout,
	}
	client := http.Client{
		Transport: &transport,
	}

	url := fmt.Sprintf("http://%s:%d", c.host, c.port)
	resp, err := client.Post(url, "text/xml", req)
	if err != nil {
		log("error", fmt.Sprintf("Something went wrong %v", err))
		return
	}
	defer resp.Body.Close()
//	status := resp.StatusCode
//    log("debug", fmt.Sprintf("executer insert, status %d, host %s:%d:%d, len %d", status, c.host,c.port,c.index, len(data)))
}

func singleSender(senders []Sender) Sender {
	// if we just have multiply senders for one host, than senders will be
	// randomly rotated
	return senders[rand.Intn(len(senders))]
}

func getSender(r string, senders []Sender) Sender {
	var senderArr []Sender
	for _, w := range senders {
		if r == w.host {
			senderArr = append(senderArr, w)
		}
	}
	randIndex := rand.Intn(len(senderArr))

	return senderArr[randIndex]
}

func sender(sender Sender, send_mon *int32) {
	log("info", fmt.Sprintf("Started sender with options: [%s:%d instance %d]", sender.host, sender.port, sender.index))
	var data_buf bytes.Buffer
	var send int32
	send = 0
	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case <-ticker:
			if data_buf.Len() > 0 {
				//	log("debug", fmt.Sprintf("sending (t) %d bytes to %s..", data_buf.Len(), sender.host))
				send_data(data_buf.String(), sender)
				atomic.AddInt32(send_mon, send)
				send = 0
				// reset buffer
				data_buf.Reset()
			}
		case input_buf := <-sender.pipe:
			fmt.Fprintf(&data_buf, "%s, ", input_buf)
			send++
		}
	}
}

func monitor(mon *Mmon, boss Boss) {
	// just pick up first sender all the time, kiss
	sender := boss.senders[0]
	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case <-ticker:
			//			log("debug", fmt.Sprintf("sending to %s..", sender.host))
			send_mon_data(atomic.SwapInt32(&mon.send, 0), atomic.SwapInt32(&mon.rcv, 0), atomic.SwapInt32(&mon.conn, 0), boss.port, sender)
		}
	}
}

func send_mon_data(m int32, r int32, c int32, port string, sender Sender) {
	ts := time.Now()
	out := fmt.Sprintf("('one_sec.int_%s.metrics_send', %d, %d, '%s'),", port, m, ts.Unix(), ts.Format("2006-01-02"))
	out = fmt.Sprintf("%s('one_sec.int_%s.metrics_rcvd', %d, %d, '%s'),", out, port, r, ts.Unix(), ts.Format("2006-01-02"))
	out = fmt.Sprintf("%s('one_sec.int_%s.conn', %d, %d, '%s')", out, port, c, ts.Unix(), ts.Format("2006-01-02"))
//	log("debug", fmt.Sprintf("MONITOR: %s", out))
	send_data(out, sender)
}

func line_reader(r *bufio.Reader, lines chan string) {
	for {
		line, _, err := r.ReadLine()
		if err != nil {
            // my oh my, such an ugly way to stop accepting data via channels :(
			lines <- "__end_of_data__"
			break
		}
		lines <- string(line)
	}
}

func process_connection(local net.Conn, boss Boss, mon *Mmon) {
	//	peer := printable_addr(local.RemoteAddr())
	//	log("debug", fmt.Sprintf("got connection from peer %s", peer))

	//parse_chan := make(chan string)
	//go parse(parse_chan, boss)

	defer local.Close()
	r := bufio.NewReader(local)
	// tag for loop to break it properly
	lines := make(chan string)
	go line_reader(r, lines)
	last_rcv := time.Now()
	ticker := time.Tick(1 * time.Second)
L:
	for {
		select {
		case line := <-lines:
			//            line := <-lines
			if line == "__end_of_data__" {
				break L
			}
			// process data
			atomic.AddInt32(&mon.rcv, 1)
			parse(line, boss)
			last_rcv = time.Now()
		case <-ticker:
			if time.Since(last_rcv).Seconds() > 60 {
				log("debug", "closing connection after read timeout 60sec")
				break L
			}
		}
	}
}

func parse(input string, boss Boss) {
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
		// convert timestamp to int64
		ts_int, err := strconv.ParseInt(ts, 0, 64)
		if err != nil {
			log("error", "Failed to parse to int")
			return
		}
		t := time.Unix(ts_int, 0)

		if boss.single == 1 {
			w := singleSender(boss.senders)
			w.pipe <- fmt.Sprintf("('%s', %s, %d, '%s')", metric, data, t.Unix(), t.Format("2006-01-02"))
		} else {
			r, err := boss.ring.GetN(metric, boss.rf)
			if err != nil {
				log("error", fmt.Sprintf("Failed to get caches for metric %s, err %v", metric, err))
				return
			}

			for _, item := range r {
				w := getSender(item, boss.senders)
				w.pipe <- fmt.Sprintf("('%s', %s, %d, '%s')", metric, data, t.Unix(), t.Format("2006-01-02"))
			}
		}
	} else {
		log("error", fmt.Sprintf("[Error] Bad formated input: %s", input))
	}
	return
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
    /*
    f, err := os.Create("cacher.prf")
	if err != nil {
		os.Exit(1)
	}
    */
    go func() {
        http.ListenAndServe("0.0.0.0:6060", nil)
    }()
//	pprof.StartCPUProfile(f)
//	defer pprof.StopCPUProfile()
	// parse config
    flag.Parse()
	var config mylib.Config
	if _, err := os.Stat(*confFile); os.IsNotExist(err) {
		fmt.Printf("no such file: %s, loading default\n", *confFile)
		config = mylib.Load("")
	} else {
		config = mylib.Load(*confFile)
        fmt.Printf("using %s as config file\n", *confFile)
	}

	// start logger
	LogC = make(chan *Msg, 100000)
	go logger_loop(LogC)
	log("info", "Starting")

	// set hash ring object
	r := consistent.New()

	// set up monitoring
	mon := new(Mmon)

	// spawn db writers and fill hash ring object
	workers := make([]Sender, 0)
	index := 0
	for _, st := range config.Storages {
		for j := 0; j <= st.Num; j++ {
			log("info", fmt.Sprintf("Starting %d worker", index))
			var w Sender
			w.host = st.Host
			w.port = st.Port
			w.pipe = make(chan string, worker_chan_size)
			w.index = index
			r.Add(st.Host)
			index++
			workers = append(workers, w)
			go sender(w, &mon.send)
		}
	}

	// create Boss var (used to hide tons of vars in functions stack)
	var boss Boss
	boss.senders = workers
	boss.rf = config.Rf
	boss.ring = r
	boss.single = 0
    boss.port = config.Port
	// if we have a single host, than we can ignore hash ring mess
	// and do simple rr rotation of senders
	if len(boss.ring.Members()) == 1 {
		boss.single = 1
	}
	rand.Seed(time.Now().Unix())

	go monitor(mon, boss)

	// start listener
	ln, err := net.Listen("tcp", ":"+config.Port)
    log("info", fmt.Sprintf("Started on %s port", config.Port))
	if err != nil {
		log("error", fmt.Sprintf("Unable to start listener, %v", err))
		os.Exit(1)
	}

	// main loop
	for {
		conn, err := ln.Accept()
		if err == nil {
			go process_connection(conn, boss, mon)
			// received new connection
			atomic.AddInt32(&mon.conn, 1)
		} else {
			log("debug", fmt.Sprintf("Failed to accept connection, %v", err))
		}
    //pprof.WriteHeapProfile(f)
	}

	log("info", "Done")
	// close log
	LogC <- &Msg{}

    //f.Close()
}
