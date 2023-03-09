// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
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
		wg.Add(3)
		go tcpClient()
		go udpClient()
		go webClient()
	} else if strings.EqualFold(mod, MOD_UDP) {
		wg.Add(1)
		go udpClient()
	} else if strings.EqualFold(mod, MOD_WEB) {
		wg.Add(1)
		go webClient()
	} else {
		wg.Add(1)
		go tcpClient()
	}

	wg.Wait()
}

func tcpClient() {
	var tcpAddr *net.TCPAddr

	tcpAddr, _ = net.ResolveTCPAddr("tcp", addr+":"+tcpPort)

	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	if err != nil {
		fmt.Println("TCP: Client connect error ! " + err.Error())
		return
	}

	defer conn.Close()

	fmt.Println(conn.LocalAddr().String() + " : TCP Client connected!")

	onMessageReceived(conn)
}

func udpClient() {
	var udpAddr *net.UDPAddr

	udpAddr, _ = net.ResolveUDPAddr("udp", addr+":"+udpPort)

	conn, err := net.DialUDP("udp", nil, udpAddr)

	if err != nil {
		fmt.Println("UDP: Client connect error ! " + err.Error())
		return
	}

	defer conn.Close()

	fmt.Println(conn.LocalAddr().String() + " : UDP Client connected!")

	onMessageReceivedUDP(conn)
}

func webClient() {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://"+addr+":"+webPort+"/", nil)
	if err != nil {
		fmt.Println("WEB: connect server failed")
		return
	}
	defer conn.Close()

	fmt.Println(conn.LocalAddr().String() + " : WEB Client connected!")

	onMessageReceivedWEB(conn)
}

func onMessageReceived(conn *net.TCPConn) {
	reader := bufio.NewReader(conn)
	b := []byte(conn.LocalAddr().String() + " Say hello to TCP Server... \n")
	_, err := conn.Write(b)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		msg, err := reader.ReadString('\n')
		fmt.Println(msg)

		if err != nil || err == io.EOF {
			fmt.Println(err)
			break
		}
		time.Sleep(time.Second * 2)

		b := []byte(conn.LocalAddr().String() + " write data to TCP Server... \n")
		_, err = conn.Write(b)

		if err != nil {
			fmt.Println(err)
			break
		}
	}
}

func onMessageReceivedUDP(conn *net.UDPConn) {
	reader := bufio.NewReader(conn)
	b := []byte(conn.LocalAddr().String() + " Say hello to UDP Server... \n")
	_, err := conn.Write(b)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		msg, err := reader.ReadString('\n')
		if err != nil || err == io.EOF {
			fmt.Println(err)
			break
		}
		fmt.Println(msg)

		time.Sleep(time.Second * 2)

		b := []byte(conn.LocalAddr().String() + " write data to UDP Server... \n")
		_, err = conn.Write(b)

		if err != nil {
			fmt.Println(err)
			break
		}
	}
}

func onMessageReceivedWEB(conn *websocket.Conn) {
	err := conn.WriteMessage(websocket.TextMessage, []byte("Say hello to Web Server... \n"))
	if nil != err {
		fmt.Println(err)
		return
	}

	for {
		_, messageData, err := conn.ReadMessage()
		if nil != err {
			fmt.Println(err)
			break
		}
		fmt.Println(string(messageData))

		time.Sleep(time.Second * 2)

		err = conn.WriteMessage(websocket.TextMessage, []byte(conn.LocalAddr().String()+" write data to Web Server... \n"))
		if nil != err {
			fmt.Println(err)
			return
		}
	}
}
