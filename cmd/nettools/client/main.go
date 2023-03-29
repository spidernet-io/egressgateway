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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var wg sync.WaitGroup

func main() {
	config := utils.ParseFlag()
	if *config.Addr == "" {
		log.Fatalln("err: server addr no provide")
	}
	protocol := strings.ToLower(*config.Proto)
	switch protocol {
	case utils.PROTOCOL_TCP:
		wg.Add(1)
		go tcpClient(config)
	case utils.PROTOCOL_UDP:
		wg.Add(1)
		go udpClient(config)
	case utils.PROTOCOL_WEB:
		wg.Add(1)
		go webClient(config)
	case utils.PROTOCOL_ALL:
		wg.Add(3)
		go tcpClient(config)
		go udpClient(config)
		go webClient(config)
	default:
		log.Fatalf("protocol: %s don't support, available protocols: tcp,udp,web,all", *config.Proto)
	}

	wg.Wait()
}

func tcpClient(config utils.Config) {
	defer wg.Done()

	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr(utils.PROTOCOL_TCP, fmt.Sprintf("%s:%s", *config.Addr, *config.TcpPort))

	log.Println("trying to connect tcpServer: ", fmt.Sprintf("%s:%s", *config.Addr, *config.TcpPort))
	conn, err := net.DialTCP(utils.PROTOCOL_TCP, nil, tcpAddr)

	if err != nil {
		log.Fatalln("WEB: connect server failed: ", err)
	}

	defer conn.Close()

	fmt.Println(conn.LocalAddr().String() + " : TCP Client connected!")

	onMessageReceived(conn)
}

func udpClient(config utils.Config) {
	defer wg.Done()

	var udpAddr *net.UDPAddr
	udpAddr, _ = net.ResolveUDPAddr(utils.PROTOCOL_UDP, fmt.Sprintf("%s:%s", *config.Addr, *config.UdpPort))

	log.Println("trying to connect udpServer: ", fmt.Sprintf("%s:%s", *config.Addr, *config.UdpPort))
	conn, err := net.DialUDP(utils.PROTOCOL_UDP, nil, udpAddr)

	if err != nil {
		log.Fatalln("WEB: connect server failed: ", err)
	}

	defer conn.Close()

	fmt.Println(conn.LocalAddr().String() + " : UDP Client connected!")

	onMessageReceivedUDP(conn)
}

func webClient(config utils.Config) {
	defer wg.Done()

	dialer := websocket.Dialer{}
	log.Println("trying to connect websocket: ", fmt.Sprintf("ws://%s:%s/", *config.Addr, *config.WebPort))
	conn, _, err := dialer.Dial(fmt.Sprintf("ws://%s:%s/", *config.Addr, *config.WebPort), nil)
	if err != nil {
		log.Fatalln("WEB: connect server failed: ", err)
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
