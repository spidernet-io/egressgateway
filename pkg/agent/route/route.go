// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package route

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

func NewRuleRoute(table int, mark, mask int, multiPath bool, log *zap.Logger) *RuleRoute {
	return &RuleRoute{
		mark:      mark,
		mask:      mask,
		table:     table,
		multiPath: multiPath,
		log:       log,
	}
}

type RuleRoute struct {
	mark      int
	mask      int
	table     int
	multiPath bool
	log       *zap.Logger
}

func (r *RuleRoute) Ensure(linkName string, ipv4List, ipv6List []net.IP) error {
	if len(ipv4List) > 0 {
		r.log.Debug("ensure rule v4")
		err := r.ensureRule(netlink.FAMILY_V4)
		if err != nil {
			return err
		}
	}
	if len(ipv6List) > 0 {
		r.log.Debug("ensure rule v6")
		err := r.ensureRule(netlink.FAMILY_V6)
		if err != nil {
			return err
		}
	}

	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return err
	}
	r.log.Sugar().Debugf("get link %s", linkName)

	if r.multiPath {
		r.log.Debug("ensure multi route v4")
		err := r.ensureMultiRoute(link, ipv4List, netlink.FAMILY_V4)
		if err != nil {
			return err
		}
		r.log.Debug("ensure multi route v4")
		err = r.ensureMultiRoute(link, ipv6List, netlink.FAMILY_V6)
		if err != nil {
			return err
		}
	} else {
		var ipv4, ipv6 *net.IP

		if len(ipv4List) > 0 {
			ipv4 = &ipv4List[0]
		}

		if len(ipv6List) > 0 {
			ipv6 = &ipv6List[0]
		}

		r.log.Debug("ensure route v4")
		err := r.ensureRoute(link, ipv4, netlink.FAMILY_V4)
		if err != nil {
			return err
		}
		r.log.Debug("ensure route v6")
		err = r.ensureRoute(link, ipv6, netlink.FAMILY_V6)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RuleRoute) ensureRoute(link netlink.Link, ip *net.IP, family int) error {
	routeFilter := &netlink.Route{Table: r.table}
	routes, err := netlink.RouteListFiltered(family, routeFilter, netlink.RT_FILTER_TABLE)
	if err != nil {
		return err
	}

	var find bool
	for _, route := range routes {
		if route.Table == r.table {
			if ip == nil || route.Gw.String() != ip.String() {
				r.log.Sugar().Infof("delete route: %s, exp ip %v", route.String(), ip)
				err := netlink.RouteDel(&route)
				if err != nil {
					return err
				}
				continue
			}
			find = true
		}
	}

	if ip == nil {
		return nil
	}

	if !find {
		index := link.Attrs().Index
		err = netlink.RouteAdd(&netlink.Route{LinkIndex: index, Gw: *ip, Table: r.table})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RuleRoute) ensureMultiRoute(link netlink.Link, ips []net.IP, family int) error {
	routeFilter := &netlink.Route{Table: r.table}
	routes, err := netlink.RouteListFiltered(family, routeFilter, netlink.RT_FILTER_TABLE)
	if err != nil {
		return err
	}
	index := link.Attrs().Index

	for _, route := range routes {
		if route.Table == r.table {

			expMap := make(map[string]net.IP, 0)
			for _, ip := range ips {
				expMap[ip.String()] = ip
			}

			for _, path := range route.MultiPath {
				if _, ok := expMap[path.Gw.String()]; !ok {
					continue
				}
				delete(expMap, path.Gw.String())
			}

			if len(expMap) > 0 {
				route := newMultiRoute(index, r.table, ips)
				err = netlink.RouteReplace(route)
				if err != nil {
					return fmt.Errorf("replace route with error: %v", err)
				}
			}

			return nil
		}
	}

	route := newMultiRoute(index, r.table, ips)
	err = netlink.RouteAdd(route)
	if err != nil {
		return fmt.Errorf("add route with error: %v", err)
	}

	return nil
}

func newMultiRoute(index, table int, ips []net.IP) *netlink.Route {
	paths := make([]*netlink.NexthopInfo, 0)
	for _, ip := range ips {
		info := &netlink.NexthopInfo{LinkIndex: index, Gw: ip}
		paths = append(paths, info)
	}
	route := &netlink.Route{Table: table, MultiPath: paths}
	return route
}

func (r *RuleRoute) ensureRule(family int) error {
	rules, err := netlink.RuleList(family)
	if err != nil {
		return err
	}
	r.log.Sugar().Debugf("number of rule: %d", len(rules))
	needAdd := true
	for _, rule := range rules {
		if rule.Table == r.table {
			r.log.Sugar().Debugf("check same table rule: %v", rule.String())

			if rule.Mask != r.mask || rule.Mark != r.mark {
				r.log.Sugar().Debugf("delete rule: %v", rule.String())
				err := netlink.RuleDel(&netlink.Rule{Table: r.table})
				if err != nil {
					return err
				}
				continue
			}
			needAdd = false
		}
	}
	r.log.Sugar().Debugf("check rule done, is need add: %v", needAdd)

	if needAdd {
		rule := netlink.NewRule()
		rule.Table = r.table
		rule.Mark = r.mark
		rule.Mask = r.mask
		rule.Family = family

		r.log.Sugar().Debugf("add rule: %v", rule.String())
		err := netlink.RuleAdd(rule)
		if err != nil {
			return err
		}
	}
	return nil
}
