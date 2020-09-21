package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
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
		broker   = flag.String("broker", "tcp://localhost:1883", "MQTT broker endpoint as scheme://host:port")
		topic    = flag.String("topic", "/test", "MQTT topic for outgoing messages")
		username = flag.String("username", "", "MQTT username (empty if auth disabled)")
		password = flag.String("password", "", "MQTT password (empty if auth disabled)")
		qos      = flag.Int("qos", 1, "QoS for published messages")
		size     = flag.Int("size", 100, "Size of the messages payload (bytes)")
		count    = flag.Int("count", 100, "Number of messages to send per client")
		clients  = flag.Int("clients", 10, "Number of clients to start")
		delay    = flag.Int("delay", 1, "Delay between messages")
		format   = flag.String("format", "text", "Output format: text|json")
		quiet    = flag.Bool("quiet", false, "Suppress logs while running")
		folderName = flag.String("name", "test", "Name of the simulation folder")
	)

	flag.Parse()
	if *clients < 1 {
		log.Fatalf("Invalid arguments: number of clients should be > 1, given: %v", *clients)
	}

	if *count < 1 {
		log.Fatalf("Invalid arguments: messages count should be > 1, given: %v", *count)
	}

	resCh := make(chan *RunResults)
	start := time.Now()
	for i := 0; i < *clients; i++ {
		if !*quiet {
			log.Println("Starting client ", i)
		}
		c := &Client{
			ID:         i,
			BrokerURL:  *broker,
			BrokerUser: *username,
			BrokerPass: *password,
			MsgTopic:   *topic,
			MsgSize:    *size,
			MsgCount:   *count,
			Delay:	    *delay,
			MsgQoS:     byte(*qos),
			Quiet:      *quiet,
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
	printResults(results, totals, start, *broker, *folderName, *format)
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

func printResults(results []*RunResults, totals *TotalResults, startPub time.Time, broker string, folder string, format string) {
	data := [][]string{}
	var resToString [7]string

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
		json.Indent(&out, data, "", "\t")

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
	
	if strings.Contains(broker, ".") {
		//remove tcp:// remove port (after :)
		broker = strings.Split(broker, ".")[2]
	}
	
	//create path. experiment/MMDD
	path := fmt.Sprintf("experiments/%v/%v", startPub.Format("0102"), folder)
	os.MkdirAll(path, os.ModePerm)

	//filename: b2_pubtime_HHmmSS 
	fmt.Printf("%v/b%v_pubtime_%v.csv", path, broker, startPub.Format("150405"))
	file, err := os.Create(fmt.Sprintf("%v/b%v_pubtime_%v.csv", path, broker, startPub.Format("150405")))
	checkError("Cannot create file", err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, res := range results{
		resToString[0] = fmt.Sprintf("%d", res.ID)
		resToString[1] = fmt.Sprintf("%.3f", res.RunTime)
		resToString[2] = fmt.Sprintf("%.3f", res.MsgTimeMin)
		resToString[3] = fmt.Sprintf("%.3f", res.MsgTimeMax)
		resToString[4] = fmt.Sprintf("%.3f", res.MsgTimeMean)
		resToString[5] = fmt.Sprintf("%.3f", res.MsgTimeStd)
		resToString[6] = fmt.Sprintf("%.3f", res.MsgsPerSec)
		data = append(data, []string{fmt.Sprintf("broker_%v", broker), resToString[0], resToString[1], resToString[2], resToString[3],
						resToString[4], resToString[5], resToString[6]})
	}

	for _, value := range data {
        	err := writer.Write(value)
		checkError("Cannot write to file", err)
	}	

	return
}

func checkError(message string, err error) {
    if err != nil {
        log.Fatal(message, err)
    }
}
