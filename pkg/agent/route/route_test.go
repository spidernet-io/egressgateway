// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package route

import (
	"errors"
	"net"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"

	"github.com/spidernet-io/egressgateway/pkg/markallocator"
)

func TestPurgeStaleRules(t *testing.T) {
	cases := map[string]struct {
		prepare func() []gomonkey.Patches
		expErr  bool
	}{
		"failed RangeSize": {
			prepare: errPurgeStaleRulesRangeSize,
			expErr:  true,
		},
		"failed RuleListFiltered v4": {
			prepare: errPurgeStaleRulesRuleListFilteredV4,
			expErr:  true,
		},
		"failed RuleListFiltered v6": {
			prepare: errPurgeStaleRulesRuleListFilteredV6,
			expErr:  true,
		},
		"failed RuleDel v4": {
			prepare: errPurgeStaleRulesRuleDelV4,
			expErr:  true,
		},
		"failed RuleDel v6": {
			prepare: errPurgeStaleRulesRuleDelV6,
			expErr:  true,
		},
		"succeed": {},
	}
	ruleRoute := NewRuleRoute()

	marks := map[int]struct{}{
		1: {},
		2: {},
	}

	baseMark := "1000"

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			patches := make([]gomonkey.Patches, 0)
			if tc.prepare != nil {
				patches = tc.prepare()
			}
			err := ruleRoute.PurgeStaleRules(marks, baseMark)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func TestEnsure(t *testing.T) {
	cases := map[string]struct {
		makePatch func(*RuleRoute) []gomonkey.Patches
		prepare   func() (string, *net.IP, *net.IP, int, int)
		expErr    bool
	}{
		"zero mark": {
			prepare: mockEnsureZeroMark,
		},
		"failed EnsureRule v4": {
			makePatch: errEnsureEnsureRuleV4,
			prepare:   mockEnsureParams,
			expErr:    true,
		},
		"failed EnsureRule v6": {
			makePatch: errEnsureEnsureRuleV6,
			prepare:   mockEnsureParams,
			expErr:    true,
		},
		"failed LinkByName": {
			prepare: mockEnsureParams,
			expErr:  true,
		},
		"failed EnsureRoute v4": {
			makePatch: errEnsureEnsureRouteV4,
			prepare:   mockEnsureParams,
			expErr:    true,
		},
		"failed EnsureRoute v6": {
			makePatch: errEnsureEnsureRouteV6,
			prepare:   mockEnsureParams,
			expErr:    true,
		},
		"succeeded Ensure": {
			prepare:   mockEnsureParams,
			makePatch: successEnsure,
		},
	}
	ruleRoute := NewRuleRoute()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			var patches = make([]gomonkey.Patches, 0)
			if tc.makePatch != nil {
				patches = tc.makePatch(ruleRoute)
			}

			name, ipv4, ipv6, table, mark := tc.prepare()
			if tc.expErr {
				err = ruleRoute.Ensure(name, ipv4, ipv6, table, mark)
				assert.Error(t, err)
			} else {
				err = ruleRoute.Ensure(name, ipv4, ipv6, table, mark)
				assert.NoError(t, err)
			}

			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func TestEnsureRoute(t *testing.T) {
	cases := map[string]struct {
		makePatch func() []gomonkey.Patches
		prepare   func() (netlink.Link, *net.IP, int, int, logr.Logger)
		expErr    bool
	}{
		"failed RouteListFiltered v4": {
			prepare:   mockEnsureRouteParams,
			makePatch: errEnsureRouteRouteListFiltered,
			expErr:    true,
		},
		"failed RouteDel v4": {
			prepare:   mockEnsureRouteParams,
			makePatch: errEnsureRouteRouteDel,
			expErr:    true,
		},
		"succeeded EnsureRoute": {
			prepare:   mockEnsureRouteParams,
			makePatch: successEnsureRoute,
		},
		"nil ip": {
			prepare:   mockEnsureRouteEmptyIP,
			makePatch: errEnsureRouteEmptyIP,
		},

		"failed RouteAdd": {
			prepare:   mockEnsureRouteParams,
			makePatch: errEnsureRouteRouteAdd,
			expErr:    true,
		},
	}
	ruleRoute := NewRuleRoute()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			var patches = make([]gomonkey.Patches, 0)
			if tc.makePatch != nil {
				patches = tc.makePatch()
			}

			name, ip, family, table, log := tc.prepare()
			if tc.expErr {
				err = ruleRoute.EnsureRoute(name, ip, family, table, log)
				assert.Error(t, err)
			} else {
				err = ruleRoute.EnsureRoute(name, ip, family, table, log)
				assert.NoError(t, err)
			}

			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func TestEnsureRule(t *testing.T) {
	cases := map[string]struct {
		makePatch func() []gomonkey.Patches
		prepare   func() (int, int, int, logr.Logger)
		expErr    bool
	}{
		"failed RuleListFiltered v4": {
			prepare:   mockEnsureRuleParams,
			makePatch: errEnsureRuleRuleListFiltered,
			expErr:    true,
		},
		"failed RuleDel": {
			prepare:   mockEnsureRuleParams,
			makePatch: errEnsureRuleRuleDel,
			expErr:    true,
		},
		"succeeded found": {
			prepare:   mockEnsureRuleParams,
			makePatch: successEnsureRuleFound,
		},
		"failed RuleAdd": {
			prepare:   mockEnsureRuleParams,
			makePatch: errEnsureRuleRuleAdd,
			expErr:    true,
		},
		"succeeded RuleAdd": {
			prepare:   mockEnsureRuleParams,
			makePatch: successEnsureRuleRuleAdd,
		},

		"succeeded multi RuleDel": {
			prepare:   mockEnsureRuleParams,
			makePatch: successEnsureRuleMultiRuleDel,
		},
	}
	ruleRoute := NewRuleRoute()

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			var patches = make([]gomonkey.Patches, 0)
			if tc.makePatch != nil {
				patches = tc.makePatch()
			}
			family, table, mark, log := tc.prepare()
			if tc.expErr {
				err = ruleRoute.EnsureRule(family, table, mark, log)
				assert.Error(t, err)
			} else {
				err = ruleRoute.EnsureRule(family, table, mark, log)
				assert.NoError(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func errPurgeStaleRulesRangeSize() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(markallocator.RangeSize, uint64(0), uint64(0), errors.New("some error"))
	return []gomonkey.Patches{*patch}
}

func errPurgeStaleRulesRuleListFilteredV4() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, nil, errors.New("some error"))
	return []gomonkey.Patches{*patch}
}

func errPurgeStaleRulesRuleListFilteredV6() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncSeq(netlink.RuleListFiltered, []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil, nil}, Times: 1},
		{Values: gomonkey.Params{nil, errors.New("some error")}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}

func errPurgeStaleRulesRuleDelV4() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{{Mark: 5000, Priority: 99}}, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.RuleDel, errors.New("some error"))
	return []gomonkey.Patches{*patch, *patch2}
}

func errPurgeStaleRulesRuleDelV6() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{{Mark: 5000, Priority: 99}}, nil)
	patch2 := gomonkey.ApplyFuncSeq(netlink.RuleDel, []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errors.New("some error")}, Times: 1},
	})
	return []gomonkey.Patches{*patch, *patch2}
}

func mockEnsureZeroMark() (string, *net.IP, *net.IP, int, int) {
	return "test-link", nil, nil, 0, 0
}

func mockEnsureParams() (string, *net.IP, *net.IP, int, int) {
	ipv4 := net.ParseIP("192.168.0.1")
	ipv6 := net.ParseIP("2001:db8::1")
	return "test-link", &ipv4, &ipv6, 1000, 1234
}

func errEnsureEnsureRuleV4(r *RuleRoute) []gomonkey.Patches {
	patch := gomonkey.ApplyMethodReturn(r, "EnsureRule", errors.New("some err"))
	return []gomonkey.Patches{*patch}
}

func errEnsureEnsureRuleV6(r *RuleRoute) []gomonkey.Patches {
	patch := gomonkey.ApplyMethodSeq(r, "EnsureRule", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errors.New("some err")}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}

func errEnsureEnsureRouteV4(r *RuleRoute) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{}, nil)
	patch := gomonkey.ApplyMethodReturn(r, "EnsureRoute", errors.New("some err"))
	return []gomonkey.Patches{*patch1, *patch}
}

func errEnsureEnsureRouteV6(r *RuleRoute) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{}, nil)
	patch := gomonkey.ApplyMethodSeq(r, "EnsureRoute", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errors.New("some err")}, Times: 1},
	})
	return []gomonkey.Patches{*patch1, *patch}
}

func successEnsure(r *RuleRoute) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{}, nil)
	patch2 := gomonkey.ApplyMethodReturn(r, "EnsureRoute", nil)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mockEnsureRouteParams() (netlink.Link, *net.IP, int, int, logr.Logger) {
	link := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Index: 1,
			Name:  "test-link",
		},
	}
	ipv4 := net.ParseIP("192.168.0.1")
	family := netlink.FAMILY_V4
	table := 1000
	log := logr.Logger{}
	return link, &ipv4, family, table, log
}

func mockEnsureRouteEmptyIP() (netlink.Link, *net.IP, int, int, logr.Logger) {
	link := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Index: 1,
			Name:  "test-link",
		},
	}
	family := netlink.FAMILY_V4
	table := 1000
	log := logr.Logger{}
	return link, nil, family, table, log
}

func errEnsureRouteRouteListFiltered() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, nil, errors.New("some error"))
	return []gomonkey.Patches{*patch}
}

func errEnsureRouteRouteDel() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, []netlink.Route{
		{Table: 1000},
	}, nil)
	patch := gomonkey.ApplyFuncReturn(netlink.RouteDel, errors.New("some error"))
	return []gomonkey.Patches{*patch, *patch1}
}

func successEnsureRoute() []gomonkey.Patches {
	gw := net.ParseIP("192.168.0.1")
	patch1 := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, []netlink.Route{
		{Table: 1000, Gw: gw},
	}, nil)
	patch := gomonkey.ApplyFuncReturn(netlink.RouteDel, nil)
	return []gomonkey.Patches{*patch, *patch1}
}

func errEnsureRouteEmptyIP() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, []netlink.Route{
		{Table: 1000},
	}, nil)
	patch := gomonkey.ApplyFuncReturn(netlink.RouteDel, nil)
	return []gomonkey.Patches{*patch, *patch1}
}

func errEnsureRouteRouteAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, []netlink.Route{
		{Table: 1000},
	}, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.RouteDel, nil)
	patch3 := gomonkey.ApplyFuncReturn(netlink.RouteAdd, errors.New("some err"))

	return []gomonkey.Patches{*patch2, *patch1, *patch3}
}

func mockEnsureRuleParams() (int, int, int, logr.Logger) {
	family := netlink.FAMILY_V4
	table := 1000
	mark := 1234
	log := logr.Logger{}
	return family, table, mark, log
}

func errEnsureRuleRuleListFiltered() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, nil, errors.New("some error"))
	return []gomonkey.Patches{*patch}
}

func errEnsureRuleRuleDel() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{
		{Table: 100},
	}, nil)

	patch := gomonkey.ApplyFuncReturn(netlink.RuleDel, errors.New("some error"))
	return []gomonkey.Patches{*patch, *patch1}
}

func successEnsureRuleFound() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{
		{Table: 1000, Priority: 99},
	}, nil)

	return []gomonkey.Patches{*patch1}
}

func errEnsureRuleRuleAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{}, nil)

	patch := gomonkey.ApplyFuncReturn(netlink.RuleAdd, errors.New("some error"))
	return []gomonkey.Patches{*patch, *patch1}
}

func successEnsureRuleRuleAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{}, nil)

	patch := gomonkey.ApplyFuncReturn(netlink.RuleAdd, nil)
	return []gomonkey.Patches{*patch, *patch1}
}

func successEnsureRuleMultiRuleDel() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{
		{Table: 1000, Priority: 99},
		{Table: 1000, Priority: 99},
	}, nil)

	patch := gomonkey.ApplyFuncSeq(netlink.RuleDel, []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{nil}, Times: 1},
	})
	return []gomonkey.Patches{*patch, *patch1}
}
