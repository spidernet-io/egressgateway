// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"bufio"
	"context"
	"fmt"
	"github.com/spidernet-io/egressgateway/cmd/nettools/flag"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

type Case func(ctx context.Context, config flag.Config) error

// Batch is used to e2e
func Batch(ctx context.Context, config flag.Config) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(*config.Timeout))
	defer cancel()

	var cases []Case

	switch *config.Proto {
	case "udp":
		cases = []Case{udp}
	case "tcp":
		cases = []Case{tcp}
	case "wss":
		cases = []Case{wss}
	default:
		cases = []Case{tcp, udp, wss}
	}

	wg := &sync.WaitGroup{}

	for _, c := range cases {
		cc := c
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			err := cc(ctx, config)
			if err != nil {
				fmt.Println(err)
				cancel()
			}
			wg.Done()
		}(wg)
	}

	wg.Wait()
	return nil
}

func tcp(ctx context.Context, config flag.Config) error {
	ip := net.ParseIP(*config.Addr)
	var addrStr string
	if ip.To4() == nil {
		addrStr = fmt.Sprintf("[%s]:%s", *config.Addr, *config.TcpPort)
	} else {
		addrStr = fmt.Sprintf("%s:%s", *config.Addr, *config.TcpPort)
	}
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addrStr)
	if err != nil {
		return fmt.Errorf("failed to connect tcp server: %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)
	b := []byte(conn.LocalAddr().String() + " Say hello to TCP Server... \n")
	_, err = conn.Write(b)
	if err != nil {
		return fmt.Errorf("failed send msg to tcp server: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cancel to check egress IP")
		default:
			msg, err := reader.ReadString('\n')

			if err != nil || err == io.EOF {
				fmt.Println(err)
				break
			}

			if strings.Contains(msg, *config.EgressIP) {
				if *config.Contain {
					return nil
				} else {
					continue
				}
			} else {
				if !*config.Contain {
					return nil
				} else {
					return fmt.Errorf("wo got egressIP")
				}
			}
		}
	}
}

func udp(ctx context.Context, config flag.Config) error {
	return nil
}

func wss(ctx context.Context, config flag.Config) error {
	return nil
}
