// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package route

import (
	"errors"
	"net"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-logr/logr"
	"github.com/spidernet-io/egressgateway/pkg/markallocator"
	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
)

var mockLogger = logr.Logger{}

func TestPurgeStaleRules(t *testing.T) {
	cases := map[string]struct {
		prepare func() []gomonkey.Patches
		expErr  bool
	}{
		"failed RangeSize": {
			prepare: err_PurgeStaleRules_RangeSize,
			expErr:  true,
		},
		"failed RuleListFiltered v4": {
			prepare: err_PurgeStaleRules_RuleListFilteredV4,
			expErr:  true,
		},
		"failed RuleListFiltered v6": {
			prepare: err_PurgeStaleRules_RuleListFilteredV6,
			expErr:  true,
		},
		"failed RuleDel v4": {
			prepare: err_PurgeStaleRules_RuleDelV4,
			expErr:  true,
		},
		"failed RuleDel v6": {
			prepare: err_PurgeStaleRules_RuleDelV6,
			expErr:  true,
		},
		"succeed": {},
	}
	ruleRoute := NewRuleRoute(mockLogger)

	marks := map[int]struct{}{
		1: {},
		2: {},
	}

	baseMark := "1000"

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			var patches = make([]gomonkey.Patches, 0)
			if tc.prepare != nil {
				patchess := tc.prepare()
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}
			if tc.expErr {
				err = ruleRoute.PurgeStaleRules(marks, baseMark)
				assert.Error(t, err)
			} else {
				err = ruleRoute.PurgeStaleRules(marks, baseMark)
				assert.NoError(t, err)
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
			prepare: mock_Ensure__zeroMark,
		},
		"failed EnsureRule v4": {
			makePatch: err_Ensure_EnsureRuleV4,
			prepare:   mock_Ensure_params,
			expErr:    true,
		},
		"failed EnsureRule v6": {
			makePatch: err_Ensure_EnsureRuleV6,
			prepare:   mock_Ensure_params,
			expErr:    true,
		},
		"failed LinkByName": {
			prepare: mock_Ensure_params,
			expErr:  true,
		},
		"failed EnsureRoute v4": {
			makePatch: err_Ensure_EnsureRouteV4,
			prepare:   mock_Ensure_params,
			expErr:    true,
		},
		"failed EnsureRoute v6": {
			makePatch: err_Ensure_EnsureRouteV6,
			prepare:   mock_Ensure_params,
			expErr:    true,
		},
		"succeeded Ensure": {
			makePatch: succ_Ensure,
			prepare:   mock_Ensure_params,
		},
	}
	ruleRoute := NewRuleRoute(mockLogger)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			var patches = make([]gomonkey.Patches, 0)
			if tc.makePatch != nil {
				patchess := tc.makePatch(ruleRoute)
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			name, ipv4, ipv6, table, mark := tc.prepare()
			if tc.expErr {
				err = ruleRoute.Ensure(name, ipv4, ipv6, table, mark)
				assert.Error(t, err)
			} else {
				err = ruleRoute.Ensure(name, ipv4, ipv6, table, mark)
				assert.NoError(t, err)
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
			prepare:   mock_EnsureRoute_params,
			makePatch: err_EnsureRoute_RouteListFiltered,
			expErr:    true,
		},
		"failed RouteDel v4": {
			prepare:   mock_EnsureRoute_params,
			makePatch: err_EnsureRoute_RouteDel,
			expErr:    true,
		},
		"succeeded EnsureRoute": {
			prepare:   mock_EnsureRoute_params,
			makePatch: succ_EnsureRoute,
		},
		"nil ip": {
			prepare:   mock_EnsureRoute_empty_ip,
			makePatch: err_EnsureRoute_empty_ip,
		},

		"failed RouteAdd": {
			prepare:   mock_EnsureRoute_params,
			makePatch: err_EnsureRoute_RouteAdd,
			expErr:    true,
		},
	}
	ruleRoute := NewRuleRoute(mockLogger)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			var patches = make([]gomonkey.Patches, 0)
			if tc.makePatch != nil {
				patchess := tc.makePatch()
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			name, ip, family, table, log := tc.prepare()
			if tc.expErr {
				err = ruleRoute.EnsureRoute(name, ip, family, table, log)
				assert.Error(t, err)
			} else {
				err = ruleRoute.EnsureRoute(name, ip, family, table, log)
				assert.NoError(t, err)
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
			prepare:   mock_EnsureRule_params,
			makePatch: err_EnsureRule_RuleListFiltered,
			expErr:    true,
		},
		"failed RuleDel": {
			prepare:   mock_EnsureRule_params,
			makePatch: err_EnsureRule_RuleDel,
			expErr:    true,
		},
		"succeeded found": {
			prepare:   mock_EnsureRule_params,
			makePatch: succ_EnsureRule_found,
		},
		"failed RuleAdd": {
			prepare:   mock_EnsureRule_params,
			makePatch: err_EnsureRule_RuleAdd,
			expErr:    true,
		},
		"succeeded RuleAdd": {
			prepare:   mock_EnsureRule_params,
			makePatch: succ_EnsureRule_RuleAdd,
		},

		"succeeded multi-RuleDel": {
			prepare:   mock_EnsureRule_params,
			makePatch: succ_EnsureRule_multi_RuleDel,
		},
	}
	ruleRoute := NewRuleRoute(mockLogger)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			var patches = make([]gomonkey.Patches, 0)
			if tc.makePatch != nil {
				patchess := tc.makePatch()
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			family, table, mark, log := tc.prepare()
			if tc.expErr {
				err = ruleRoute.EnsureRule(family, table, mark, log)
				assert.Error(t, err)
			} else {
				err = ruleRoute.EnsureRule(family, table, mark, log)
				assert.NoError(t, err)
			}
		})
	}
}

func err_PurgeStaleRules_RangeSize() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(markallocator.RangeSize, uint64(0), uint64(0), errors.New("some error"))
	return []gomonkey.Patches{*patch}
}

func err_PurgeStaleRules_RuleListFilteredV4() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, nil, errors.New("some error"))
	return []gomonkey.Patches{*patch}
}

func err_PurgeStaleRules_RuleListFilteredV6() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncSeq(netlink.RuleListFiltered, []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil, nil}, Times: 1},
		{Values: gomonkey.Params{nil, errors.New("some error")}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}

func err_PurgeStaleRules_RuleDelV4() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{{Mark: 5000}}, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.RuleDel, errors.New("some error"))
	return []gomonkey.Patches{*patch, *patch2}
}

func err_PurgeStaleRules_RuleDelV6() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{{Mark: 5000}}, nil)
	patch2 := gomonkey.ApplyFuncSeq(netlink.RuleDel, []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errors.New("some error")}, Times: 1},
	})
	return []gomonkey.Patches{*patch, *patch2}
}

func mock_Ensure__zeroMark() (string, *net.IP, *net.IP, int, int) {
	return "testlink", nil, nil, 0, 0
}

func mock_Ensure_params() (string, *net.IP, *net.IP, int, int) {
	ipv4 := net.ParseIP("192.168.0.1")
	ipv6 := net.ParseIP("2001:db8::1")
	return "testlink", &ipv4, &ipv6, 1000, 1234
}

func err_Ensure_EnsureRuleV4(r *RuleRoute) []gomonkey.Patches {
	patch := gomonkey.ApplyMethodReturn(r, "EnsureRule", errors.New("some err"))
	return []gomonkey.Patches{*patch}
}

func err_Ensure_EnsureRuleV6(r *RuleRoute) []gomonkey.Patches {
	patch := gomonkey.ApplyMethodSeq(r, "EnsureRule", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errors.New("some err")}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}

func err_Ensure_EnsureRouteV4(r *RuleRoute) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{}, nil)
	patch := gomonkey.ApplyMethodReturn(r, "EnsureRoute", errors.New("some err"))
	return []gomonkey.Patches{*patch1, *patch}
}

func err_Ensure_EnsureRouteV6(r *RuleRoute) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{}, nil)
	patch := gomonkey.ApplyMethodSeq(r, "EnsureRoute", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errors.New("some err")}, Times: 1},
	})
	return []gomonkey.Patches{*patch1, *patch}
}

func succ_Ensure(r *RuleRoute) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Dummy{}, nil)
	patch := gomonkey.ApplyMethodReturn(r, "EnsureRoute", nil)
	return []gomonkey.Patches{*patch1, *patch}
}

func mock_EnsureRoute_params() (netlink.Link, *net.IP, int, int, logr.Logger) {
	link := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Index: 1,
			Name:  "testlink",
		},
	}
	ipv4 := net.ParseIP("192.168.0.1")
	family := 4
	table := 1000
	log := logr.Logger{}
	return link, &ipv4, family, table, log
}

func mock_EnsureRoute_empty_ip() (netlink.Link, *net.IP, int, int, logr.Logger) {
	link := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Index: 1,
			Name:  "testlink",
		},
	}
	family := 4
	table := 1000
	log := logr.Logger{}
	return link, nil, family, table, log
}

func err_EnsureRoute_RouteListFiltered() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, nil, errors.New("some error"))
	return []gomonkey.Patches{*patch}
}

func err_EnsureRoute_RouteDel() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, []netlink.Route{
		{Table: 1000},
	}, nil)
	patch := gomonkey.ApplyFuncReturn(netlink.RouteDel, errors.New("some error"))
	return []gomonkey.Patches{*patch, *patch1}
}

func succ_EnsureRoute() []gomonkey.Patches {
	gw := net.ParseIP("192.168.0.1")
	patch1 := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, []netlink.Route{
		{Table: 1000, Gw: gw},
	}, nil)
	patch := gomonkey.ApplyFuncReturn(netlink.RouteDel, nil)
	return []gomonkey.Patches{*patch, *patch1}
}

func err_EnsureRoute_empty_ip() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, []netlink.Route{
		{Table: 1000},
	}, nil)
	patch := gomonkey.ApplyFuncReturn(netlink.RouteDel, nil)
	return []gomonkey.Patches{*patch, *patch1}
}

func err_EnsureRoute_RouteAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RouteListFiltered, []netlink.Route{
		{Table: 1000},
	}, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.RouteDel, nil)
	patch3 := gomonkey.ApplyFuncReturn(netlink.RouteAdd, errors.New("some err"))

	return []gomonkey.Patches{*patch2, *patch1, *patch3}
}

func mock_EnsureRule_params() (int, int, int, logr.Logger) {
	family := 4
	table := 1000
	mark := 1234
	log := logr.Logger{}
	return family, table, mark, log
}

func err_EnsureRule_RuleListFiltered() []gomonkey.Patches {
	patch := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, nil, errors.New("some error"))
	return []gomonkey.Patches{*patch}
}

func err_EnsureRule_RuleDel() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{
		{Table: 100},
	}, nil)

	patch := gomonkey.ApplyFuncReturn(netlink.RuleDel, errors.New("some error"))
	return []gomonkey.Patches{*patch, *patch1}
}

func succ_EnsureRule_found() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{
		{Table: 1000},
	}, nil)

	return []gomonkey.Patches{*patch1}
}

func err_EnsureRule_RuleAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{}, nil)

	patch := gomonkey.ApplyFuncReturn(netlink.RuleAdd, errors.New("some error"))
	return []gomonkey.Patches{*patch, *patch1}
}

func succ_EnsureRule_RuleAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{}, nil)

	patch := gomonkey.ApplyFuncReturn(netlink.RuleAdd, nil)
	return []gomonkey.Patches{*patch, *patch1}
}

func succ_EnsureRule_multi_RuleDel() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.RuleListFiltered, []netlink.Rule{
		{Table: 1000},
		{Table: 1000},
	}, nil)

	patch := gomonkey.ApplyFuncSeq(netlink.RuleDel, []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{nil}, Times: 1},
	})
	return []gomonkey.Patches{*patch, *patch1}
}
