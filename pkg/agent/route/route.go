// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package route

import (
	"net"

	"github.com/go-logr/logr"
	"github.com/vishvananda/netlink"

	"github.com/spidernet-io/egressgateway/pkg/markallocator"
)

func NewRuleRoute(log logr.Logger) *RuleRoute {
	return &RuleRoute{log: log}
}

type RuleRoute struct {
	log logr.Logger
}

func (r *RuleRoute) PurgeStaleRules(marks map[int]struct{}, baseMark string) error {
	start, end, err := markallocator.RangeSize(baseMark)
	if err != nil {
		return err
	}

	clean := func(rules []netlink.Rule, family int) error {
		for _, rule := range rules {
			rule.Family = family
			if _, ok := marks[rule.Mark]; !ok {
				if int(start) <= rule.Mark && int(end) >= rule.Mark {
					err := netlink.RuleDel(&rule)
					if err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	rules, err := netlink.RuleListFiltered(netlink.FAMILY_V4, nil, netlink.RT_FILTER_MARK)
	if err != nil {
		return err
	}
	if err := clean(rules, netlink.FAMILY_V4); err != nil {
		return err
	}

	rules, err = netlink.RuleListFiltered(netlink.FAMILY_V6, nil, netlink.RT_FILTER_MARK)
	if err != nil {
		return err
	}
	if err := clean(rules, netlink.FAMILY_V6); err != nil {
		return err
	}

	return nil
}

func (r *RuleRoute) Ensure(linkName string, ipv4, ipv6 *net.IP, table int, mark int) error {
	if mark == 0 {
		return nil
	}

	log := r.log.WithValues("linkName", linkName, "table", table, "mark", mark)

	if ipv4 != nil {
		err := r.ensureRule(netlink.FAMILY_V4, table, mark, log)
		if err != nil {
			return err
		}
	}

	if ipv6 != nil {
		err := r.ensureRule(netlink.FAMILY_V6, table, mark, log)
		if err != nil {
			return err
		}
	}

	link, err := netlink.LinkByName(linkName)
	if err != nil {
		return err
	}

	log.V(1).Info("get link")

	err = r.ensureRoute(link, ipv4, netlink.FAMILY_V4, table, log)
	if err != nil {
		return err
	}
	err = r.ensureRoute(link, ipv6, netlink.FAMILY_V6, table, log)
	if err != nil {
		return err
	}
	return nil
}

func (r *RuleRoute) ensureRoute(link netlink.Link, ip *net.IP, family int, table int, log logr.Logger) error {
	log = log.WithValues("family", family, "ip", ip)
	log.V(1).Info("ensure route")

	routeFilter := &netlink.Route{Table: table}
	routes, err := netlink.RouteListFiltered(family, routeFilter, netlink.RT_FILTER_TABLE)
	if err != nil {
		return err
	}

	var find bool
	for _, route := range routes {
		if route.Table == table {
			if ip == nil || route.Gw.String() != ip.String() {
				log.Info("delete route", "route", route.String())
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
		err = netlink.RouteAdd(&netlink.Route{LinkIndex: index, Gw: *ip, Table: table})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RuleRoute) ensureRule(family int, table int, mark int, log logr.Logger) error {
	log = log.WithValues("family", family)
	log.V(1).Info("ensure rule")

	t := netlink.NewRule()
	t.Mark = mark
	t.Family = family
	rules, err := netlink.RuleListFiltered(family, t, netlink.RT_FILTER_MARK)
	if err != nil {
		return err
	}
	r.log.V(1).Info("list rule", "count", len(rules))

	found := false
	for _, rule := range rules {
		del := false
		if rule.Table != table {
			del = true
		}
		if found {
			del = true
		}
		if del {
			rule.Family = family
			err = netlink.RuleDel(&rule)
			if err != nil {
				return err
			}
			continue
		}
		found = true
	}
	if found {
		return nil
	}

	if !found {
		r.log.V(1).Info("rule not match, try add it")
		rule := netlink.NewRule()
		rule.Table = table
		rule.Mark = mark
		rule.Family = family

		r.log.V(1).Info("add rule", "rule", rule.String())
		err := netlink.RuleAdd(rule)
		if err != nil {
			return err
		}
	}
	return nil
}
