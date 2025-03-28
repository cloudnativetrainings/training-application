package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/magiconair/properties"
	log "github.com/sirupsen/logrus"
)

var alive = true
var ready = true

type Config struct {
	Message string `properties:"message"`
}

var props *properties.Properties

func init() {
	props = properties.MustLoadFile("./conf/app.conf", properties.UTF8)
}

func main() {

	go handleStdin()
	go handleLifecycle()

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/liveness", handleLiveness)
	http.HandleFunc("/readiness", handleReadiness)
	http.HandleFunc("/downward_api", handleDownwardApi)
	http.HandleFunc("/cats", handleCats)

	log.Info("App started")
	http.ListenAndServe(":8080", nil)
}

func handleStdin() {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)
		if text != "" {
			handleCommand(text)
		}
	}
}

func handleCommand(command string) {
	if command == "set ready" {
		log.Info("Set application to ready")
		ready = true
	} else if command == "set unready" {
		log.Info("Set application to unready")
		ready = false
	} else if command == "set alive" {
		log.Info("Set application to alive")
		alive = true
	} else if command == "set dead" {
		log.Info("Set application to dead")
		alive = false
	} else if command == "leak mem" {
		log.Info("Leaking Memory")
		leakMem()
	} else if command == "leak cpu" {
		log.Info("Leaking CPU")
		leakCpu()
	} else if strings.HasPrefix(command, "request ") {
		url, _ := strings.CutPrefix(command, "request ")
		request(url)
	} else {
		log.Infof("Unknown command '%s'", command)
	}
}

func request(url string) {
	log.Infof("Request '%s'", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Errorf("Error on getting the response: '%s'", err)
	}
	defer resp.Body.Close()

	log.Infof("StatusCode of response %d", resp.StatusCode)

	if resp.TLS == nil {
		log.Info("Response is not encrypted")
	} else {
		log.Info("Response is encrypted")
		log.Infof("TLS Version: %d", resp.TLS.Version)
		for _, cert := range resp.TLS.PeerCertificates {
			log.Infof("Certificate Subject: %s", cert.Subject.String())
			log.Infof("Certificate Issuer: %s", cert.Issuer.String())
			log.Infof("Certificate Serial Number: %s", cert.SerialNumber.String())
			log.Infof("Certificate Not Before: %s", cert.NotBefore.String())
			log.Infof("Certificate Not After: %s", cert.NotAfter.String())
			log.Infof("Certificate DNS Names: %v", cert.DNSNames)
			log.Infof("Certificate Email Addresses: %v", cert.EmailAddresses)
			log.Infof("Certificate IP Addresses: %v", cert.IPAddresses)
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error on reading the response body: '%s'", err)
	}
	bodyString := string(bodyBytes)
	if len(bodyString) >= 100 {
		bodyString = bodyString[:100]
	}
	log.Infof("Response Body: %s", bodyString)
}

func handleCats(w http.ResponseWriter, r *http.Request) {

	type catStruct struct {
		Url string `json:"url"`
	}

	resp, err := http.Get("https://api.thecatapi.com/v1/images/search")
	if err != nil {
		log.Errorf("Error on getting the response: '%s'", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error on reading the response body: '%s'", err)
	}
	bodyString := string(bodyBytes)
	log.Infof("Got response from cat api: %s", bodyString)

	var cats []catStruct
	json.Unmarshal(bodyBytes, &cats)
	cat := cats[0].Url

	fmt.Fprintf(w, "<!DOCTYPE html><htlml><body>")
	fmt.Fprintf(w, "<img src='%s'></img>", cat)
	fmt.Fprintf(w, "</body></htlml>")
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	message := props.GetString("message", "Property is not set")
	fmt.Fprintf(w, "<!DOCTYPE html><htlml><body>")
	fmt.Fprintf(w, "Message: %s<br>", message)
	fmt.Fprintf(w, "Pod Name: %s<br>", os.Getenv("POD_NAME"))
	fmt.Fprintf(w, "Pod IP: %s<br>", os.Getenv("POD_IP"))
	fmt.Fprintf(w, "Live: %t<br>", alive)
	fmt.Fprintf(w, "Ready: %t<br>", ready)
	fmt.Fprintf(w, "</body></htlml>")
}

func handleLiveness(w http.ResponseWriter, r *http.Request) {
	if alive {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	if ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func handleDownwardApi(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "MY_NODE_NAME: %s<br>", os.Getenv("MY_NODE_NAME"))
	fmt.Fprintf(w, "MY_POD_NAME: %s<br>", os.Getenv("MY_POD_NAME"))
	fmt.Fprintf(w, "MY_POD_IP: %s<br>", os.Getenv("MY_POD_IP"))
}

func handleLifecycle() {
	signalChanel := make(chan os.Signal, 1)
	signal.Notify(signalChanel, syscall.SIGTERM)
	exitChanel := make(chan int)
	go func() {
		for {
			s := <-signalChanel
			if s == syscall.SIGTERM {
				log.Info("Got SIGTERM signal")
				log.Info("Starting Graceful Shutdown")
				for i := 0; i < 10; i++ {
					log.Infof("Graceful shutdown took %d seconds", i)
					time.Sleep(1 * time.Second)
				}
				log.Info("Graceful Shutdown has finished")
				exitChanel <- 0
			} else {
				log.Info("Got unknown signal")
				exitChanel <- 1
			}
		}
	}()
	exitCode := <-exitChanel
	os.Exit(exitCode)
}

func leakMem() {
	memLeak := make([]string, 0)
	count := 0
	for {
		if count%1000 == 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			fmt.Printf("Alloc = %v MiB", m.Alloc/1024/1024)
			fmt.Printf("\tTotalAlloc = %v MiB", m.TotalAlloc/1024/1024)
			fmt.Printf("\tSys = %v MiB", m.Sys/1024/1024)
			fmt.Printf("\tNumGC = %v\n", m.NumGC)
		}
		time.Sleep(time.Nanosecond)
		count++
		memLeak = append(memLeak, "THIS IS A MEM LEAK")
	}
}

func leakCpu() {

	// TODO is this really the smartest way to create a CPU leak?

	f, err := os.Open(os.DevNull)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)

	for i := 0; i < n; i++ {
		go func() {
			for {
				fmt.Fprintf(f, ".")
			}
		}()
	}

	time.Sleep(10 * time.Second)

}
