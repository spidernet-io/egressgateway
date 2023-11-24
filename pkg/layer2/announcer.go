// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

// This code is copied from the metallb project, which is also licensed under
// the Apache License, Version 2.0. The original code can be found at:
// https://github.com/metallb/metallb
// SPDX-License-Identifier:Apache-2.0

// Changes:
// * replace logger library

package layer2

import (
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/go-logr/logr"

	"github.com/spidernet-io/egressgateway/pkg/lock"
)

// Announce is used to "announce" new IPs mapped to the node's MAC address.
type Announce struct {
	logger logr.Logger

	lock.RWMutex
	nodeInterfaces []string // current local interfaces' name list
	arps           map[int]*arpResponder
	ndps           map[int]*ndpResponder
	ips            map[string][]IPAdvertisement // svcName -> IPAdvertisements
	ipRefcnt       map[string]int               // ip.String() -> number of uses

	// This channel can block - do not write to it while holding the mutex
	// to avoid deadlocking.
	spamCh        chan IPAdvertisement
	excludeRegexp *regexp.Regexp
}

// New returns an initialized Announce.
func New(l logr.Logger, excludeRegexp *regexp.Regexp) (*Announce, error) {
	ret := &Announce{
		logger:         l,
		nodeInterfaces: []string{},
		arps:           map[int]*arpResponder{},
		ndps:           map[int]*ndpResponder{},
		ips:            map[string][]IPAdvertisement{},
		ipRefcnt:       map[string]int{},
		spamCh:         make(chan IPAdvertisement, 1024),
		excludeRegexp:  excludeRegexp,
	}

	go ret.interfaceScan()
	go ret.spamLoop()

	return ret, nil
}

func (a *Announce) interfaceScan() {
	for {
		a.updateInterfaces()
		time.Sleep(10 * time.Second)
	}
}

// updateInterfaces is used to scan network interfaces, update the arps and ndps lists in
// the Announcement object to respond to ARP and NDP requests, and exclude some unnecessary
// interfaces.
func (a *Announce) updateInterfaces() {
	ifs, err := net.Interfaces()
	if err != nil {
		a.logger.Error(err, "couldn't list interfaces", "op", "getInterfaces")
		return
	}

	a.Lock()
	defer a.Unlock()

	keepARP, keepNDP := map[int]bool{}, map[int]bool{}
	curIfs := make([]string, 0, len(ifs))
	for _, intf := range ifs {
		ifi := intf

		if (a.excludeRegexp != nil) && a.excludeRegexp.MatchString(ifi.Name) {
			a.logger.V(1).Info("announced interface to exclude interface", "interface", ifi.Name)
			continue
		}

		curIfs = append(curIfs, ifi.Name)

		l := a.logger.WithValues("interface", ifi.Name)

		addrs, err := ifi.Addrs()
		if err != nil {
			l.Error(err, "couldn't get addresses for interface", "op", "getAddresses")
			return
		}

		if ifi.Flags&net.FlagUp == 0 {
			continue
		}
		if _, err = os.Stat("/sys/class/net/" + ifi.Name + "/master"); !os.IsNotExist(err) {
			continue
		}
		f, err := os.ReadFile("/sys/class/net/" + ifi.Name + "/flags")
		if err == nil {
			flags, _ := strconv.ParseUint(string(f)[:len(string(f))-1], 0, 32)
			// NOARP flag
			if flags&0x80 != 0 {
				continue
			}
		}
		if ifi.Flags&net.FlagBroadcast != 0 {
			keepARP[ifi.Index] = true
		}

		for _, a := range addrs {
			ipaddr, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ipaddr.IP.To4() != nil || !ipaddr.IP.IsLinkLocalUnicast() {
				continue
			}
			keepNDP[ifi.Index] = true
			break
		}

		if keepARP[ifi.Index] && a.arps[ifi.Index] == nil {
			resp, err := newARPResponder(a.logger, &ifi, a.shouldAnnounce)
			if err != nil {
				l.Error(err, "failed to create ARP responder", "op", "createARPResponder")
				continue
			}
			a.arps[ifi.Index] = resp
			l.Info("created ARP responder for interface", "event", "createARPResponder")
		}
		if keepNDP[ifi.Index] && a.ndps[ifi.Index] == nil {
			resp, err := newNDPResponder(a.logger, &ifi, a.shouldAnnounce)
			if err != nil {
				l.Error(err, "failed to create NDP responder", "op", "createNDPResponder")
				continue
			}
			a.ndps[ifi.Index] = resp
			l.Info("created NDP responder for interface", "event", "createNDPResponder")
		}
	}

	a.nodeInterfaces = curIfs

	for i, client := range a.arps {
		if !keepARP[i] {
			client.Close()
			delete(a.arps, i)
			a.logger.Info("deleted ARP responder for interface", "event", "deleteARPResponder", "interface", client.Interface())
		}
	}
	for i, client := range a.ndps {
		if !keepNDP[i] {
			client.Close()
			delete(a.ndps, i)
			a.logger.Info("deleted NDP responder for interface", "event", "deleteNDPResponder", "interface", client.Interface())
		}
	}
}

// spamLoop is used to send gratuitous ARP or NDP in the network to avoid ARP/NDP
// cache issues, and to periodically stop spam to maintain network performance.
func (a *Announce) spamLoop() {
	// Map IP to spam stop time.
	type timedSpam struct {
		until time.Time
		IPAdvertisement
	}
	m := map[string]timedSpam{}
	// We can't create a stopped ticker, so create one with a big period to avoid ticking for nothing
	ticker := time.NewTicker(time.Hour)
	ticker.Stop()
	for {
		select {
		case s := <-a.spamCh:
			if len(m) == 0 {
				// See https://github.com/metallb/metallb/issues/172 for the 1100 choice.
				ticker.Reset(1100 * time.Millisecond)
			}
			ipStr := s.ip.String()
			_, ok := m[ipStr]
			// Set spam stop time to 5 seconds from now.
			m[ipStr] = timedSpam{time.Now().Add(5 * time.Second), s}
			if !ok {
				// Spam right away to avoid waiting up to 1100 milliseconds even if
				// it means we call gratuitous() twice in a row in a short amount of time.
				a.gratuitous(s)
			}
		case now := <-ticker.C:
			for ipStr, tSpam := range m {
				if now.After(tSpam.until) {
					// We have spammed enough - remove the IP from the map.
					delete(m, ipStr)
				} else {
					a.gratuitous(tSpam.IPAdvertisement)
				}
			}
			if len(m) == 0 {
				ticker.Stop()
			}
		}
	}
}

func (a *Announce) doSpam(adv IPAdvertisement) {
	a.spamCh <- adv
}

// gratuitous sends arp packets
//  1. Acquire a read lock for the Announce object.
//  2. Check whether the Announce object still holds control over the given IP address.
//     If not, return without further action.
//  3. Iterate through the arps or ndps list in the Announce object, depending on the IP
//     address type (IPv4 or IPv6), and send gratuitous ARP or NDP messages for the given
//     IP address on each network interface.
//  4. If an error occurs during the sending process, log it.
//  5. Finally, release the read lock for the Announce object.
func (a *Announce) gratuitous(adv IPAdvertisement) {
	a.RLock()
	defer a.RUnlock()

	ip := adv.ip
	if a.ipRefcnt[ip.String()] <= 0 {
		// We've lost control of the IP, someone else is
		// doing announcements.
		return
	}

	if ip.To4() != nil {
		for _, client := range a.arps {
			if !adv.matchInterface(client.intf) {
				a.logger.V(1).Info("skip interfaces", "op", "gratuitousAnnounce", "interface", client.intf)
				continue
			}
			if err := client.Gratuitous(ip); err != nil {
				a.logger.Error(err, "failed to make gratuitous ARP announcement",
					"op", "gratuitousAnnounce", "ip", ip)
			}
		}
	} else {
		for _, client := range a.ndps {
			if !adv.matchInterface(client.intf) {
				a.logger.V(1).Info("skip interfaces", "op", "gratuitousAnnounce", "interface", client.intf)
				continue
			}
			if err := client.Gratuitous(ip); err != nil {
				a.logger.Error(err, "failed to make gratuitous NDP announcement",
					"op", "gratuitousAnnounce", "ip", ip)
			}
		}
	}
}

func (a *Announce) shouldAnnounce(ip net.IP, intf string) dropReason {
	a.RLock()
	defer a.RUnlock()
	ipFound := false
	for _, ipAdvertisements := range a.ips {
		for _, i := range ipAdvertisements {
			if i.ip.Equal(ip) {
				ipFound = true
				if i.matchInterface(intf) {
					return dropReasonNone
				}
			}
		}
	}
	if ipFound {
		return dropReasonNotMatchInterface
	}
	return dropReasonAnnounceIP
}

// SetBalancer adds ip to the set of announced addresses.
func (a *Announce) SetBalancer(name string, adv IPAdvertisement) {
	// Call doSpam at the end of the function without holding the lock
	defer a.doSpam(adv)
	a.Lock()
	defer a.Unlock()

	// Kubernetes may inform us that we should advertise this address multiple
	// times, so just no-op any subsequent requests.
	if ipAdvertisements, ok := a.ips[name]; ok {
		for i := range ipAdvertisements {
			if adv.ip.Equal(a.ips[name][i].ip) {
				a.ips[name][i] = adv // override in case the interface list changed
				return
			}
		}
	}
	a.ips[name] = append(a.ips[name], adv)

	a.ipRefcnt[adv.ip.String()]++
	if a.ipRefcnt[adv.ip.String()] > 1 {
		// Multiple services are using this IP, so there's nothing
		// else to do right now.
		return
	}

	for _, client := range a.ndps {
		if err := client.Watch(adv.ip); err != nil {
			a.logger.Error(err, "failed to watch NDP multicast group for IP, NDP responder will not respond to requests for this address",
				"op", "watchMulticastGroup", "ip", adv.ip, "interface", client.intf,
			)
		}
	}
}

// DeleteBalancer deletes an address from the set of addresses we should announce.
func (a *Announce) DeleteBalancer(name string) {
	a.Lock()
	defer a.Unlock()

	advs, ok := a.ips[name]
	if !ok {
		return
	}
	delete(a.ips, name)

	for _, cur := range advs {
		a.ipRefcnt[cur.ip.String()]--
		if a.ipRefcnt[cur.ip.String()] > 0 {
			// Another service is still using this IP, don't touch any
			// more things.
			continue
		}

		for _, client := range a.ndps {
			if err := client.Unwatch(cur.ip); err != nil {
				a.logger.Error(err, "failed to unwatch NDP multicast group for IP",
					"op", "unwatchMulticastGroup", "ip", cur.ip, "interface", client.intf,
				)
			}
		}
	}
}

// AnnounceName returns true when we have an announcement under name.
func (a *Announce) AnnounceName(name string) bool {
	a.RLock()
	defer a.RUnlock()
	_, ok := a.ips[name]
	return ok
}

// GetInterfaces returns current interfaces list.
func (a *Announce) GetInterfaces() []string {
	a.Lock()
	defer a.Unlock()

	localInterfaces := make([]string, len(a.nodeInterfaces))
	copy(localInterfaces, a.nodeInterfaces)
	return localInterfaces
}

// dropReason is the reason why a layer2 protocol packet was not
// responded to.
type dropReason int

// Various reasons why a packet was dropped.
const (
	dropReasonNone dropReason = iota
	dropReasonClosed
	dropReasonError
	dropReasonARPReply
	dropReasonMessageType
	dropReasonNoSourceLL
	dropReasonEthernetDestination
	dropReasonAnnounceIP
	dropReasonNotMatchInterface
)
