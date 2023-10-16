// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spidernet-io/egressgateway/cmd/nettools/flag"
)

var (
	wg       sync.WaitGroup
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func main() {
	config := flag.ParseServerFlag()
	protocol := strings.ToLower(*config.Proto)
	switch protocol {
	case flag.ProtocolTcp:
		wg.Add(1)
		go tcpServer(config)
	case flag.ProtocolUdp:
		wg.Add(1)
		go udpServer(config)
	case flag.ProtocolWeb:
		wg.Add(1)
		go websocketServer(config)
	case flag.ProtocolAll:
		wg.Add(3)
		go tcpServer(config)
		go udpServer(config)
		go websocketServer(config)
	default:
		log.Fatalf("protocol: %s don't support, available protocols: tcp,udp,web,all", *config.Proto)
	}

	wg.Wait()
}

func tcpServer(config flag.ServerConfig) {
	defer wg.Done()
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr(flag.ProtocolTcp, fmt.Sprintf("%s:%s", *config.Addr, *config.TcpPort))
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatalf("tcpServer failed to start: %v", err)
	}
	defer tcpListener.Close()
	log.Println("TCP Server listen on: ", tcpAddr.String())
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("A client connected :" + tcpConn.RemoteAddr().String())
		go tcpPipe(tcpConn)
	}

}

func tcpPipe(conn *net.TCPConn) {
	ipStr := conn.RemoteAddr().String()

	defer func() {
		fmt.Println(" Disconnected : " + ipStr)
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	i := 0
	for {
		message, err := reader.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		fmt.Println(string(message))

		time.Sleep(time.Second * 2)

		msg := time.Now().String() + " clientIP=" + conn.RemoteAddr().String() + " TCP Server Say hello! \n"

		b := []byte(msg)

		_, err = conn.Write(b)
		if err != nil {
			fmt.Println(err)
			return
		}

		i++

		if i > 100 {
			break
		}
	}
}

func udpServer(config flag.ServerConfig) {
	defer wg.Done()
	udpConn, err := net.ListenPacket(flag.ProtocolUdp, fmt.Sprintf("%s:%s", *config.Addr, *config.UdpPort))
	if err != nil {
		log.Fatalf("udpServer failed to start: %v", err)
	}
	defer udpConn.Close()
	log.Println("UDP Server listen on: ", udpConn.LocalAddr().String())
	for {
		data := make([]byte, 4096)
		read, remoteAddr, err := udpConn.ReadFrom(data)
		if err != nil {
			fmt.Println("UDP: read meg failed", err)
			continue
		}
		fmt.Println(read, remoteAddr)
		fmt.Printf("%s\n", data)

		go func() {
			senddata := []byte(time.Now().String() + " clientIP=" + remoteAddr.String() + " UDP Server Say hello! \n")
			_, err = udpConn.WriteTo(senddata, remoteAddr)
			if err != nil {
				fmt.Println("UDP: send meg failed!", err)
				return
			}
		}()
	}
}

func websocketServer(config flag.ServerConfig) {
	defer wg.Done()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		for {
			// Read message from browser
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Print the message to the console
			fmt.Println(msgType, conn.RemoteAddr())
			fmt.Println(string(msg))

			// Write message back to browser
			senddata := []byte(time.Now().String() + " clientIP=" + conn.RemoteAddr().String() + " WebSocket Server Say hello! \n")
			if err = conn.WriteMessage(msgType, senddata); err != nil {
				return
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "Remote IP: %v\n", r.RemoteAddr)
	})

	log.Println("WebSocket Server listen on: ", *config.Addr, ":", *config.WebPort)

	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", *config.Addr, *config.WebPort), nil); err != nil {
		log.Fatalf("WebSocket create failed: %v ", err)
	}
}
