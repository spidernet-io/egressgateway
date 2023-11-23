// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

// This code is copied from the metallb project, which is also licensed under
// the Apache License, Version 2.0. The original code can be found at:
// https://github.com/metallb/metallb
// SPDX-License-Identifier:Apache-2.0

package layer2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/go-logr/logr"
	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
)

type announceFunc func(net.IP, string) dropReason

type arpResponder struct {
	logger       logr.Logger
	intf         string
	hardwareAddr net.HardwareAddr
	conn         *arp.Client
	closed       chan struct{}
	announce     announceFunc
}

func newARPResponder(logger logr.Logger, ifi *net.Interface, ann announceFunc) (*arpResponder, error) {
	client, err := arp.Dial(ifi)
	if err != nil {
		return nil, fmt.Errorf("creating ARP responder for %q: %s", ifi.Name, err)
	}

	ret := &arpResponder{
		logger:       logger,
		intf:         ifi.Name,
		hardwareAddr: ifi.HardwareAddr,
		conn:         client,
		closed:       make(chan struct{}),
		announce:     ann,
	}
	go ret.run()
	return ret, nil
}

func (a *arpResponder) Interface() string { return a.intf }

func (a *arpResponder) Close() error {
	close(a.closed)
	return a.conn.Close()
}

func (a *arpResponder) Gratuitous(ip net.IP) error {
	for _, op := range []arp.Operation{arp.OperationRequest, arp.OperationReply} {
		pkt, err := arp.NewPacket(op, a.hardwareAddr, ip, ethernet.Broadcast, ip)
		if err != nil {
			return fmt.Errorf("assembling %q gratuitous packet for %q: %s", op, ip, err)
		}
		if err = a.conn.WriteTo(pkt, ethernet.Broadcast); err != nil {
			return fmt.Errorf("writing %q gratuitous packet for %q: %s", op, ip, err)
		}
		stats.SentGratuitous(ip.String())
	}
	return nil
}

func (a *arpResponder) run() {
	for a.processRequest() != dropReasonClosed {
	}
}

func (a *arpResponder) processRequest() dropReason {
	pkt, eth, err := a.conn.Read()
	if err != nil {
		// ARP listener doesn't cleanly return EOF when closed, so we
		// need to hook into the call to arpResponder.Close()
		// independently.
		select {
		case <-a.closed:
			return dropReasonClosed
		default:
		}
		if errors.Is(err, io.EOF) {
			return dropReasonClosed
		}
		return dropReasonError
	}

	// Ignore ARP replies.
	if pkt.Operation != arp.OperationRequest {
		return dropReasonARPReply
	}

	// Ignore ARP requests which are not broadcast or bound directly for this machine.
	if !bytes.Equal(eth.Destination, ethernet.Broadcast) && !bytes.Equal(eth.Destination, a.hardwareAddr) {
		return dropReasonEthernetDestination
	}

	// Ignore ARP requests that the announcer tells us to ignore.
	reason := a.announce(pkt.TargetIP, a.intf)
	if reason == dropReasonNotMatchInterface {
		a.logger.V(1).Info("ignore ARP requests",
			"op", "arpRequestIgnore", "ip", pkt.TargetIP, "interface", a.intf, "reason", "notMatchInterface")
	}
	if reason != dropReasonNone {
		return reason
	}

	stats.GotRequest(pkt.TargetIP.String())
	a.logger.V(1).Info("got ARP request for service IP, sending response",
		"interface", a.intf,
		"ip", pkt.TargetIP,
		"senderIP", pkt.SenderIP,
		"senderMAC", pkt.SenderHardwareAddr,
		"responseMAC", a.hardwareAddr,
	)

	if err := a.conn.Reply(pkt, a.hardwareAddr, pkt.TargetIP); err != nil {
		a.logger.Error(err, "failed to send ARP reply",
			"op", "arpReply",
			"interface", a.intf,
			"ip", pkt.TargetIP,
			"senderIP", pkt.SenderIP,
			"senderMAC", pkt.SenderHardwareAddr,
			"responseMAC", a.hardwareAddr,
		)
	} else {
		stats.SentResponse(pkt.TargetIP.String())
	}
	return dropReasonNone
}
