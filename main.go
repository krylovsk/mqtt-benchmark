package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/GaryBoone/GoStats/stats"
)

// Message describes a message
type Message struct {
	Topic     string
	QoS       byte
	Payload   interface{}
	Sent      time.Time
	Delivered time.Time
	Error     bool
}

// RunResults describes results of a single client / run
type RunResults struct {
	ID          int     `json:"id"`
	Successes   int64   `json:"successes"`
	Failures    int64   `json:"failures"`
	RunTime     float64 `json:"run_time"`
	MsgTimeMin  float64 `json:"msg_time_min"`
	MsgTimeMax  float64 `json:"msg_time_max"`
	MsgTimeMean float64 `json:"msg_time_mean"`
	MsgTimeStd  float64 `json:"msg_time_std"`
	MsgsPerSec  float64 `json:"msgs_per_sec"`
}

// TotalResults describes results of all clients / runs
type TotalResults struct {
	Ratio           float64 `json:"ratio"`
	Successes       int64   `json:"successes"`
	Failures        int64   `json:"failures"`
	TotalRunTime    float64 `json:"total_run_time"`
	AvgRunTime      float64 `json:"avg_run_time"`
	MsgTimeMin      float64 `json:"msg_time_min"`
	MsgTimeMax      float64 `json:"msg_time_max"`
	MsgTimeMeanAvg  float64 `json:"msg_time_mean_avg"`
	MsgTimeMeanStd  float64 `json:"msg_time_mean_std"`
	TotalMsgsPerSec float64 `json:"total_msgs_per_sec"`
	AvgMsgsPerSec   float64 `json:"avg_msgs_per_sec"`
}

// JSONResults are used to export results as a JSON document
type JSONResults struct {
	Runs   []*RunResults `json:"runs"`
	Totals *TotalResults `json:"totals"`
}

func main() {
	var (
		broker       = flag.String("broker", "tcp://localhost:1883", "MQTT broker endpoint as scheme://host:port")
		topic        = flag.String("topic", "/test", "MQTT topic for outgoing messages")
    payload      = flag.String("payload", "", "MQTT message payload. If empty, then payload is generated based on the size parameter")
		username     = flag.String("username", "", "MQTT client username (empty if auth disabled)")
		password     = flag.String("password", "", "MQTT client password (empty if auth disabled)")
		qos          = flag.Int("qos", 1, "QoS for published messages")
		wait         = flag.Int("wait", 60000, "QoS 1 wait timeout in milliseconds")
		size         = flag.Int("size", 100, "Size of the messages payload (bytes)")
		count        = flag.Int("count", 100, "Number of messages to send per client")
		clients      = flag.Int("clients", 10, "Number of clients to start")
		format       = flag.String("format", "text", "Output format: text|json")
		quiet        = flag.Bool("quiet", false, "Suppress logs while running")
		clientPrefix = flag.String("client-prefix", "mqtt-benchmark", "MQTT client id prefix (suffixed with '-<client-num>'")
		clientCert   = flag.String("client-cert", "", "Path to client certificate in PEM format")
		clientKey    = flag.String("client-key", "", "Path to private clientKey in PEM format")
	)

	flag.Parse()
	if *clients < 1 {
		log.Fatalf("Invalid arguments: number of clients should be > 1, given: %v", *clients)
	}

	if *count < 1 {
		log.Fatalf("Invalid arguments: messages count should be > 1, given: %v", *count)
	}

	if *clientCert != "" && *clientKey == "" {
		log.Fatal("Invalid arguments: private clientKey path missing")
	}

	if *clientCert == "" && *clientKey != "" {
		log.Fatalf("Invalid arguments: certificate path missing")
	}

	var tlsConfig *tls.Config
	if *clientCert != "" && *clientKey != "" {
		tlsConfig = generateTLSConfig(*clientCert, *clientKey)
	}

	resCh := make(chan *RunResults)
	start := time.Now()
	for i := 0; i < *clients; i++ {
		if !*quiet {
			log.Println("Starting client ", i)
		}
		c := &Client{
			ID:          i,
			ClientID:    *clientPrefix,
			BrokerURL:   *broker,
			BrokerUser:  *username,
			BrokerPass:  *password,
			MsgTopic:    *topic,
      MsgPayload:  *payload,
			MsgSize:     *size,
			MsgCount:    *count,
			MsgQoS:      byte(*qos),
			Quiet:       *quiet,
			WaitTimeout: time.Duration(*wait) * time.Millisecond,
			TLSConfig:   tlsConfig,
		}
		go c.Run(resCh)
	}

	// collect the results
	results := make([]*RunResults, *clients)
	for i := 0; i < *clients; i++ {
		results[i] = <-resCh
	}
	totalTime := time.Now().Sub(start)
	totals := calculateTotalResults(results, totalTime, *clients)

	// print stats
	printResults(results, totals, *format)
}

func calculateTotalResults(results []*RunResults, totalTime time.Duration, sampleSize int) *TotalResults {
	totals := new(TotalResults)
	totals.TotalRunTime = totalTime.Seconds()

	msgTimeMeans := make([]float64, len(results))
	msgsPerSecs := make([]float64, len(results))
	runTimes := make([]float64, len(results))
	bws := make([]float64, len(results))

	totals.MsgTimeMin = results[0].MsgTimeMin
	for i, res := range results {
		totals.Successes += res.Successes
		totals.Failures += res.Failures
		totals.TotalMsgsPerSec += res.MsgsPerSec

		if res.MsgTimeMin < totals.MsgTimeMin {
			totals.MsgTimeMin = res.MsgTimeMin
		}

		if res.MsgTimeMax > totals.MsgTimeMax {
			totals.MsgTimeMax = res.MsgTimeMax
		}

		msgTimeMeans[i] = res.MsgTimeMean
		msgsPerSecs[i] = res.MsgsPerSec
		runTimes[i] = res.RunTime
		bws[i] = res.MsgsPerSec
	}
	totals.Ratio = float64(totals.Successes) / float64(totals.Successes+totals.Failures)
	totals.AvgMsgsPerSec = stats.StatsMean(msgsPerSecs)
	totals.AvgRunTime = stats.StatsMean(runTimes)
	totals.MsgTimeMeanAvg = stats.StatsMean(msgTimeMeans)
	// calculate std if sample is > 1, otherwise leave as 0 (convention)
	if sampleSize > 1 {
		totals.MsgTimeMeanStd = stats.StatsSampleStandardDeviation(msgTimeMeans)
	}

	return totals
}

func printResults(results []*RunResults, totals *TotalResults, format string) {
	switch format {
	case "json":
		jr := JSONResults{
			Runs:   results,
			Totals: totals,
		}
		data, err := json.Marshal(jr)
		if err != nil {
			log.Fatalf("Error marshalling results: %v", err)
		}
		var out bytes.Buffer
		_ = json.Indent(&out, data, "", "\t")

		fmt.Println(string(out.Bytes()))
	default:
		for _, res := range results {
			fmt.Printf("======= CLIENT %d =======\n", res.ID)
			fmt.Printf("Ratio:               %.3f (%d/%d)\n", float64(res.Successes)/float64(res.Successes+res.Failures), res.Successes, res.Successes+res.Failures)
			fmt.Printf("Runtime (s):         %.3f\n", res.RunTime)
			fmt.Printf("Msg time min (ms):   %.3f\n", res.MsgTimeMin)
			fmt.Printf("Msg time max (ms):   %.3f\n", res.MsgTimeMax)
			fmt.Printf("Msg time mean (ms):  %.3f\n", res.MsgTimeMean)
			fmt.Printf("Msg time std (ms):   %.3f\n", res.MsgTimeStd)
			fmt.Printf("Bandwidth (msg/sec): %.3f\n\n", res.MsgsPerSec)
		}
		fmt.Printf("========= TOTAL (%d) =========\n", len(results))
		fmt.Printf("Total Ratio:                 %.3f (%d/%d)\n", totals.Ratio, totals.Successes, totals.Successes+totals.Failures)
		fmt.Printf("Total Runtime (sec):         %.3f\n", totals.TotalRunTime)
		fmt.Printf("Average Runtime (sec):       %.3f\n", totals.AvgRunTime)
		fmt.Printf("Msg time min (ms):           %.3f\n", totals.MsgTimeMin)
		fmt.Printf("Msg time max (ms):           %.3f\n", totals.MsgTimeMax)
		fmt.Printf("Msg time mean mean (ms):     %.3f\n", totals.MsgTimeMeanAvg)
		fmt.Printf("Msg time mean std (ms):      %.3f\n", totals.MsgTimeMeanStd)
		fmt.Printf("Average Bandwidth (msg/sec): %.3f\n", totals.AvgMsgsPerSec)
		fmt.Printf("Total Bandwidth (msg/sec):   %.3f\n", totals.TotalMsgsPerSec)
	}
	return
}

func generateTLSConfig(certFile string, keyFile string) *tls.Config {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Error reading certificate files: %v", err)
	}

	cfg := tls.Config{
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	return &cfg
}
