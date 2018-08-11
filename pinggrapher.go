package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// use a map instead of slice cause otherwise it's a pain to delete them when
// they decide to leave
var clients = make(map[int]Client)

type Client struct {
	Writer  *wsutil.Writer
	Encoder *json.Encoder
	Conn    net.Conn
}

func read(delay int, pings chan float64) {
	ticker := time.NewTicker(time.Duration(delay) * 1000 * 1000 * time.Nanosecond)
	defer ticker.Stop()
	go func() {
		var line string
		var err error
		reader := bufio.NewReader(os.Stdin)
		for {
			line, err = reader.ReadString('\n')
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Fatal(err)
			}
			ping, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
			if err != nil {
				log.Printf("Couldn't convert '%s': %s", line, err)
			}
			pings <- ping
		}
	}()
	defer func() {
		log.Println("Close all connections")
		for _, client := range clients {
			client.Conn.Close()
		}
	}()
	for {
		ping := <-pings
		for id, client := range clients {
			if err := client.Encoder.Encode(ping); err != nil {
				log.Print("Couldn't encode/write:", err)
				delete(clients, id)
			}
			if err := client.Writer.Flush(); err != nil {
				log.Print("Couldn't flush:", err)
				delete(clients, id)
			}
		}
	}
}

func startserver(port int, pings chan float64) {
	fmt.Printf("listening on :%d\n", port)
	var clientidcount = 0
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("New client.")
		conn, _, _, err := ws.UpgradeHTTP(r, w, nil)
		if err != nil {
			log.Print(err)
			return
		}
		clientidcount += 1
		writer := wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText)
		clients[clientidcount] = Client{
			Writer:  writer,
			Encoder: json.NewEncoder(writer),
			Conn:    conn,
		}
	})
	http.Handle("/", http.FileServer(http.Dir("./web")))
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func main() {
	var port int
	var delay int
	flag.IntVar(&port, "port", 9998, "port to use")
	flag.IntVar(&delay, "delay", 10000, "mms to wait before sending the data")
	flag.Parse()
	var pings = make(chan float64)
	go read(delay, pings)
	startserver(port, pings)
}
