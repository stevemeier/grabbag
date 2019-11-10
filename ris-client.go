// RIS Client for https://ris-live.ripe.net/manual/
// based on:
// https://github.com/gorilla/websocket/blob/master/examples/echo/client.go

// +build ignore

package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"

	"encoding/json"
	"github.com/davecgh/go-spew/spew"
)

type ris_message struct {
	Type string `json:"type"`
	Data struct {
		Timestamp     float64 `json:"timestamp"`
		Peer          string  `json:"peer"`
		PeerAsn       string  `json:"peer_asn"`
		ID            string  `json:"id"`
		Host          string  `json:"host"`
		Type          string  `json:"type"`
		Path          []int   `json:"path"`
		Community     [][]int `json:"community"`
		Origin        string  `json:"origin"`
		Announcements []struct {
			NextHop  string   `json:"next_hop"`
			Prefixes []string `json:"prefixes"`
		} `json:"announcements"`
	} `json:"data"`
}

var addr = flag.String("addr", "ris-live.ripe.net", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: *addr, Path: "/v1/ws/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// Tell the server what we want
	// See list of hosts at:
	// https://www.ripe.net/analyse/internet-measurements/routing-information-service-ris/ris-raw-data
	c.WriteMessage(1, []byte(`{"type": "ris_subscribe", "data": {"host": "rrc00"}}`) )

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
//			log.Printf("recv: %s", message)
			process_message(message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func process_message (message []byte) (bool) {
//	log.Printf("recv: %s", message)
	var rismessage ris_message
	if err := json.Unmarshal(message, &rismessage); err != nil {
		return false
	}
//	log.Printf("recv: %s", message)
	spew.Dump(rismessage)
	return true
}
