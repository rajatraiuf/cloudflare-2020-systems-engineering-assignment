package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Struct for parsed URL
type Httpaddr struct {
	host string
	path string
}

// Struct for profiling statistic
type Stats struct {
	statusCode int
	timeTaken  int
	respSize   int
}

func call(target Httpaddr) (string, int) {
	timeout, _ := time.ParseDuration("5s")
	dialer := net.Dialer{
		Timeout: timeout,
	}
	conn, err := tls.DialWithDialer(&dialer, "tcp", target.host+":https", nil)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}

	defer conn.Close()

	conn.Write([]byte("GET " + target.path + " HTTP/1.0\r\nHost: " + target.host + "\r\n\r\n"))

	resp, err := ioutil.ReadAll(conn)
	response := string(resp)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	respCode, _ := strconv.Atoi(response[9:12])
	conn.Close()
	return response, respCode
}

func routine(target Httpaddr, channel chan Stats, isProfiling bool) {
	start := time.Now()
	resp, statusCode := call(target)
	// Keep redirecting until a 400 or 200 status received
	if statusCode >= 300 && statusCode < 400 {
		idx := strings.Index(resp, "Location: ")
		idx1 := strings.Index(resp[idx+10:], "\n")
		go routine(parseURL(resp[idx+10:idx+10+idx1]), channel, isProfiling)
	} else {
		if !isProfiling {
			fmt.Printf("------------------------Response------------------------\n"+
				"%s\n--------------------------------------------------------\n", resp)
		}
		channel <- Stats{statusCode, int(time.Since(start).Milliseconds()), len(resp)}
	}
}

func handleTasks(URL string, isProfiling bool, count int) {
	channel, timesum := make(chan Stats, count), 0
	respTimes, respStatus, respSizes := make([]int, 0), make([]int, 0), make([]int, 0)
	target, tmp := parseURL(URL), Stats{}
	for i := 0; i < count; i++ {
		go routine(target, channel, isProfiling)
		tmp = <-channel
		respTimes, respStatus, respSizes = append(respTimes, tmp.timeTaken),
			append(respStatus, tmp.statusCode), append(respSizes, tmp.respSize)
		timesum += tmp.timeTaken
	}
	close(channel)
	if isProfiling {
		notSuccessCodes := make([]int, 0)
		for _, s := range respStatus {
			if s != 200 {
				notSuccessCodes = append(notSuccessCodes, s)
			}
		}
		sort.Ints(respTimes)
		sort.Ints(respSizes)
		fmt.Printf("------------------------Profiling Stats------------------------\n")
		fmt.Printf("Number of requests: %d\n", count)
		fmt.Printf("Fastest Time: %d\n", respTimes[0])
		fmt.Printf("Slowest Time: %d\n", respTimes[count-1])
		fmt.Printf("Mean Time: %d\n", timesum/count)
		fmt.Printf("Median Time: %d\n", respTimes[int(math.Floor(float64(len(respTimes))/2.0))])
		fmt.Printf("Percent of successful requests: %.f%%\n", float64((count-len(notSuccessCodes))/count)*100)
		fmt.Printf("Error Codes: %v\n", notSuccessCodes)
		fmt.Printf("Size in bytes of the smallest response: %d\n", respSizes[0])
		fmt.Printf("Size in bytes of the largest response: %d\n", respSizes[len(respSizes)-1])
		fmt.Printf("---------------------------------------------------------------\n")

	}
}

func parseURL(rawURL string) Httpaddr {
	r := regexp.MustCompile("^(https://|http://)")
	rawURL = r.ReplaceAllString(strings.ToLower(rawURL), "")
	host, path := rawURL, "/"
	index := strings.Index(rawURL, "/")
	if index != -1 {
		path = host[index:]
		host = host[:index]
	}
	return Httpaddr{host, path}
}

func main() {
	args := os.Args[1:]
	helpmsg := "Usage:\n\t\tgo run tool --url <Url>\n\t\tgo run tool --url <Url> --profile <Number of requests>"
	errmsg := "Error: Invalid/Missing  Arguments\n" + helpmsg
	if len(args) > 0 {
		if args[0] == "--help" {
			println(helpmsg)
		} else if args[0] == "--url" {
			if len(args) < 2 {
				println("Error: URL missing\n" + helpmsg)
			} else if len(args) > 2 && (args[2] != "--profile" || len(args) == 3) {
				println(errmsg)
			} else {
				isValidURL, _ := regexp.MatchString(`^(?:http(s)?:\/\/)?[\w.-]+(?:\.[\w\.-]+)+[\w\-\._~:/?#[\]@!\$&'\(\)\*\+,;=.]+$`,
					strings.ToLower(args[1]))
				if !isValidURL {
					print("Error: Invalid <Url>\n" + helpmsg)
					os.Exit(0)
				}
				if len(args) == 4 {
					input, e := strconv.Atoi(args[3])
					if e != nil || input < 1 {
						print("Error: Invalid <Number of requests>\n" + helpmsg)
						os.Exit(0)
					}
					handleTasks(args[1], true, input)
				} else {
					handleTasks(args[1], false, 1)
				}
			}
		} else { println(errmsg) } } else { println(errmsg)	}
}
