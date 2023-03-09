// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	MOD_TCP = "tcp"
	MOD_UDP = "udp"
	MOD_ALL = "all"
	MOD_WEB = "web"
)

var (
	addr, tcpPort, udpPort, webPort string
	wg                              sync.WaitGroup
	upgrader                        = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func main() {
	addr = os.Getenv("SERVER_IP")
	if addr == "" {
		fmt.Println("err: server addr is nil")
		return
	}
	tcpPort = os.Getenv("TCP_PORT")
	if tcpPort == "" {
		tcpPort = "8080"
	}
	udpPort = os.Getenv("UDP_PORT")
	if udpPort == "" {
		udpPort = "8081"
	}
	webPort = os.Getenv("WEB_PORT")
	if webPort == "" {
		webPort = "8082"
	}

	mod := os.Getenv("MOD")
	if strings.EqualFold(mod, MOD_ALL) {
		wg.Add(2)
		go tcpServer()
		go udpServer()
		go websocketServer()
	} else if strings.EqualFold(mod, MOD_UDP) {
		wg.Add(1)
		go udpServer()
	} else if strings.EqualFold(mod, MOD_WEB) {
		wg.Add(1)
		go websocketServer()
	} else {
		wg.Add(1)
		go tcpServer()
	}

	wg.Wait()
}

func tcpServer() {
	defer wg.Done()
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", addr+":"+tcpPort)
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer tcpListener.Close()
	fmt.Println("TCP Server listen ", tcpAddr.String())
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

func udpServer() {
	defer wg.Done()
	var udpAddr *net.UDPAddr
	udpAddr, _ = net.ResolveUDPAddr("udp", addr+":"+udpPort)
	udpListener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer udpListener.Close()
	fmt.Println("UDP Server listen ", udpAddr.String())
	for {
		data := make([]byte, 4096)
		read, remoteAddr, err := udpListener.ReadFromUDP(data)
		if err != nil {
			fmt.Println("UDP: read meg failed", err)
			continue
		}
		fmt.Println(read, remoteAddr)
		fmt.Printf("%s\n", data)

		senddata := []byte(time.Now().String() + " clientIP=" + remoteAddr.String() + " UDP Server Say hello! \n")
		_, err = udpListener.WriteToUDP(senddata, remoteAddr)
		if err != nil {
			fmt.Println("UDP: send meg failed!", err)
			return
		}
	}
}

func websocketServer() {
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

	fmt.Println("WebSocket Server listen ", addr, ":", webPort)

	if err := http.ListenAndServe(addr+":"+webPort, nil); err != nil {
		fmt.Println("WebSocket create failed; ", err)
		return
	}
}
