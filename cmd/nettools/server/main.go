// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"fmt"
	utils "github.com/spidernet-io/egressgateway/cmd/nettools"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	wg       sync.WaitGroup
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func main() {
	config := utils.ParseFlag()
	protocol := strings.ToLower(*config.Proto)
	switch protocol {
	case utils.PROTOCOL_TCP:
		wg.Add(1)
		go tcpServer(config)
	case utils.PROTOCOL_UDP:
		wg.Add(1)
		go udpServer(config)
	case utils.PROTOCOL_WEB:
		wg.Add(1)
		go websocketServer(config)
	case utils.PROTOCOL_ALL:
		wg.Add(3)
		go tcpServer(config)
		go udpServer(config)
		go websocketServer(config)
	default:
		log.Fatalf("protocol: %s don't support, available protocols: tcp,udp,web,all", *config.Proto)
	}

	wg.Wait()
}

func tcpServer(config utils.Config) {
	defer wg.Done()
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr(utils.PROTOCOL_TCP, fmt.Sprintf("%s:%s", *config.Addr, *config.TcpPort))
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

func udpServer(config utils.Config) {
	defer wg.Done()
	udpConn, err := net.ListenPacket(utils.PROTOCOL_UDP, fmt.Sprintf("%s:%s", *config.Addr, *config.UdpPort))
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

func websocketServer(config utils.Config) {
	defer wg.Done()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	http.ServeFile(w, r, "websockets.html")
	// })

	log.Println("WebSocket Server listen on: ", *config.Addr, ":", *config.WebPort)

	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", *config.Addr, *config.WebPort), nil); err != nil {
		log.Fatalf("WebSocket create failed: %v ", err)
	}
}
