package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr,
			"usage: %s <interval> <interesting_stats_regex> [<statsd_endpoint>]\n",
			path.Base(os.Args[0]))
		os.Exit(1)
	}

	interval, err := time.ParseDuration(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid interval: %s\n", err)
		os.Exit(2)
	}

	interesting_stats_regexp, err := regexp.Compile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid interesting stats expression: %s\n", err)
		os.Exit(3)
	}

	var statsd io.Writer
	if len(os.Args) == 4 {
		var err error
		statsd, err = net.Dial("udp", os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create UDP connection: %s\n", err)
			os.Exit(4)
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unknown hostname: %s\n", err)
		os.Exit(5)
	}
	hfields := strings.Split(hostname, ".")
	prefix := hfields[0]

	memcache, err := net.Dial("tcp", "127.0.0.1:11211")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to local memcache: %s\n", err)
		os.Exit(6)
	}
	reader := bufio.NewReader(memcache)

	for {
		fmt.Fprintf(memcache, "stats\r\n")
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to read from local memcache: %s\n", err)
				os.Exit(7)
			}
			text := strings.TrimSpace(line)
			if text == "END" {
				break
			}
			// fmt.Println(text)
			info := strings.Fields(text)
			if info[0] != "STAT" {
				fmt.Fprintf(os.Stderr, "Unexpected STAT info: %s\n", text)
				os.Exit(8)
			}
			if interesting_stats_regexp.MatchString(info[1]) {
				report := fmt.Sprintf("%s.memcache.%s:%s|g", prefix, info[1], info[2])
				fmt.Println(report)
				if statsd != nil {
					_, err := statsd.Write([]byte(report))
					if err != nil {
						fmt.Fprintf(os.Stderr, "Unable to write stat: %s\n", err)
						os.Exit(9)
					}
				}
			}
		}

		time.Sleep(interval)
	}
}
