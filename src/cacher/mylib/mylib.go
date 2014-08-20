package mylib

import (
    "code.google.com/p/gcfg"
    "fmt"
    "os"
    "strings"
    "strconv"
    "net"
    "time"
)

type Storage struct {
    Host string
    Port int
    Num int
}

type Config struct {
    LogLevel string
    Storage string
    Storages []*Storage
    ChanLimit int
    Rf int
    EnableDelta int
}

type configFile struct {
    Main Config
}

const defaultConfig = `
    [main]
    logLevel = 10
    storage = 127.0.0.1:8123:1
    Rf = 1
`

func parseStorages (storage string) []*Storage {
    stArr := strings.Split(storage,",")
    storages := make([]*Storage,0 )
    for _, val := range stArr {
        if len(val) < 5 {
            fmt.Printf("storage definition must be atleast 5 chars: a:1:1, %s\n", val)
            continue
        }
        val = strings.Trim(val," ")
        stOpts := strings.Split(val, ":")
        if len(stOpts) < 3 {
            fmt.Printf("Bad format for storage, not enought options (host:port:num): %s\n", val)
            os.Exit(1)
        }
        host := stOpts[0]
        port,err := strconv.ParseInt(stOpts[1],0,32)
        if err != nil {
            fmt.Printf("Bad format for storage definition, port must be int, %s\n", val)
            os.Exit(1)
        }
        num,err := strconv.ParseInt(stOpts[2],0,32)
        if err != nil {
            fmt.Println("Bad format for storage definition, num must be int, %s\n", val)
            os.Exit(1)
        }
        // convert 0,1,2.. to 1,2,3
        num = num - 1
        storages = append(storages, &Storage{host, int(port), int(num)})
    }
    return storages
}

func Load(cfgFile string) Config {
    var err error
    var cfg configFile

    if cfgFile != "" {
        err = gcfg.ReadFileInto(&cfg, cfgFile)
    } else {
        err = gcfg.ReadStringInto(&cfg, defaultConfig)
    }
    if err != nil {
        fmt.Printf("Failed to init config, %v\n", err)
        os.Exit(1)
    }

    cfg.Main.Storages = parseStorages(cfg.Main.Storage)
    return cfg.Main
}

func DialTimeout(network, addr string) (net.Conn, error) {
	c, err := net.DialTimeout(network, addr, time.Duration(60*time.Second))
	if err != nil {
		return c, err
	}
	c.SetDeadline(time.Now().Add(time.Duration(60 * time.Second)))
	return c, err
}


