// Package server tests for Basil web server.
//
// This file tests request context building functionality in handler.go
// (buildRequestContext function that populates basil.http.request).
package server

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sambeau/basil/server/config"
)

func TestQueryToMapFlags(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		expect map[string]interface{}
	}{
		{
			name:   "flag without value",
			raw:    "flag",
			expect: map[string]interface{}{"flag": true},
		},
		{
			name:   "flag explicit empty",
			raw:    "flag=",
			expect: map[string]interface{}{"flag": ""},
		},
		{
			name: "mixed flags",
			raw:  "a&b=1&c",
			expect: map[string]interface{}{
				"a": true,
				"b": "1",
				"c": true,
			},
		},
		{
			name: "mixed multi value",
			raw:  "flag=1&flag",
			expect: map[string]interface{}{
				"flag": []interface{}{"1", true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := queryToMap(tt.raw)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Fatalf("queryToMap(%q) = %#v, want %#v", tt.raw, got, tt.expect)
			}
		})
	}
}

func TestBuildRequestContextAddsRoute(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/reports/2025", nil)
	req = req.WithContext(withSubpath(req.Context(), "/2025"))

	ctx := buildRequestContext(req, config.Route{})

	if _, ok := ctx["subpath"]; ok {
		t.Fatalf("expected subpath to be absent")
	}

	routeVal, ok := ctx["route"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected route map, got %T", ctx["route"])
	}

	if routeVal["__type"] != "path" {
		t.Fatalf("expected __type path, got %v", routeVal["__type"])
	}

	segs, ok := routeVal["segments"].([]interface{})
	if !ok {
		t.Fatalf("expected segments slice, got %T", routeVal["segments"])
	}

	if len(segs) != 1 || segs[0] != "2025" {
		t.Fatalf("expected segments [2025], got %#v", segs)
	}
}
