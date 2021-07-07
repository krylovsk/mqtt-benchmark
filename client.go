package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/GaryBoone/GoStats/stats"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client implements an MQTT client running benchmark test
type Client struct {
	ID          int
	ClientID    string
	BrokerURL   string
	BrokerUser  string
	BrokerPass  string
	MsgTopic    string
	MsgPayload  string
	MsgSize     int
	MsgCount    int
	MsgQoS      byte
	Quiet       bool
	WaitTimeout time.Duration
	TLSConfig   *tls.Config
}

// Run runs benchmark tests and writes results in the provided channel
func (c *Client) Run(res chan *RunResults) {
	newMsgs := make(chan *Message)
	pubMsgs := make(chan *Message)
	doneGen := make(chan bool)
	donePub := make(chan bool)
	runResults := new(RunResults)

	started := time.Now()
	// start generator
	go c.genMessages(newMsgs, doneGen)
	// start publisher
	go c.pubMessages(newMsgs, pubMsgs, doneGen, donePub)

	runResults.ID = c.ID
	times := []float64{}
	for {
		select {
		case m := <-pubMsgs:
			if m.Error {
				log.Printf("CLIENT %v ERROR publishing message: %v: at %v\n", c.ID, m.Topic, m.Sent.Unix())
				runResults.Failures++
			} else {
				// log.Printf("Message published: %v: sent: %v delivered: %v flight time: %v\n", m.Topic, m.Sent, m.Delivered, m.Delivered.Sub(m.Sent))
				runResults.Successes++
				times = append(times, m.Delivered.Sub(m.Sent).Seconds()*1000) // in milliseconds
			}
		case <-donePub:
			// calculate results
			duration := time.Since(started)
			runResults.MsgTimeMin = stats.StatsMin(times)
			runResults.MsgTimeMax = stats.StatsMax(times)
			runResults.MsgTimeMean = stats.StatsMean(times)
			runResults.RunTime = duration.Seconds()
			runResults.MsgsPerSec = float64(runResults.Successes) / duration.Seconds()
			// calculate std if sample is > 1, otherwise leave as 0 (convention)
			if c.MsgCount > 1 {
				runResults.MsgTimeStd = stats.StatsSampleStandardDeviation(times)
			}

			// report results and exit
			res <- runResults
			return
		}
	}
}

func (c *Client) genMessages(ch chan *Message, done chan bool) {
	var payload interface{}
	for i := 0; i < c.MsgCount; i++ {
		// set payload if specified
		if c.MsgPayload != "" {
			payload = c.MsgPayload
		} else {
			payload = make([]byte, c.MsgSize)
		}

		ch <- &Message{
			Topic:   c.MsgTopic,
			QoS:     c.MsgQoS,
			Payload: payload,
		}
	}

	done <- true
	// log.Printf("CLIENT %v is done generating messages\n", c.ID)
}

func (c *Client) pubMessages(in, out chan *Message, doneGen, donePub chan bool) {
	onConnected := func(client mqtt.Client) {
		if !c.Quiet {
			log.Printf("CLIENT %v is connected to the broker %v\n", c.ID, c.BrokerURL)
		}
		ctr := 0
		for {
			select {
			case m := <-in:
				m.Sent = time.Now()
				token := client.Publish(m.Topic, m.QoS, false, m.Payload)
				res := token.WaitTimeout(c.WaitTimeout)
				if !res {
					log.Printf("CLIENT %v Timeout sending message: %v\n", c.ID, token.Error())
					m.Error = true
				} else if token.Error() != nil {
					log.Printf("CLIENT %v Error sending message: %v\n", c.ID, token.Error())
					m.Error = true
				} else {
					m.Delivered = time.Now()
					m.Error = false
				}
				out <- m

				if ctr > 0 && ctr%100 == 0 {
					if !c.Quiet {
						log.Printf("CLIENT %v published %v messages and keeps publishing...\n", c.ID, ctr)
					}
				}
				ctr++
			case <-doneGen:
				donePub <- true
				if !c.Quiet {
					log.Printf("CLIENT %v is done publishing\n", c.ID)
				}
				return
			}
		}
	}

	opts := mqtt.NewClientOptions().
		AddBroker(c.BrokerURL).
		SetClientID(fmt.Sprintf("%s-%v", c.ClientID, c.ID)).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetOnConnectHandler(onConnected).
		SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			log.Printf("CLIENT %v lost connection to the broker: %v. Will reconnect...\n", c.ID, reason.Error())
		})
	if c.BrokerUser != "" && c.BrokerPass != "" {
		opts.SetUsername(c.BrokerUser)
		opts.SetPassword(c.BrokerPass)
	}
	if c.TLSConfig != nil {
		opts.SetTLSConfig(c.TLSConfig)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	if token.Error() != nil {
		log.Printf("CLIENT %v had error connecting to the broker: %v\n", c.ID, token.Error())
	}
}
