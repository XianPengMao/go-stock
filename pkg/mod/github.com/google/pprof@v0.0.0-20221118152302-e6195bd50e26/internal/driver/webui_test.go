// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"sync"
	"testing"

	"github.com/google/pprof/internal/plugin"
	"github.com/google/pprof/internal/proftest"
	"github.com/google/pprof/profile"
)

func makeTestServer(t testing.TB, prof *profile.Profile) *httptest.Server {
	if runtime.GOOS == "nacl" || runtime.GOOS == "js" {
		t.Skip("test assumes tcp available")
	}

	// Custom http server creator
	var server *httptest.Server
	serverCreated := make(chan bool)
	creator := func(a *plugin.HTTPServerArgs) error {
		server = httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if h := a.Handlers[r.URL.Path]; h != nil {
					h.ServeHTTP(w, r)
				}
			}))
		serverCreated <- true
		return nil
	}

	// Start server and wait for it to be initialized
	go serveWebInterface("unused:1234", prof, &plugin.Options{
		Obj:        fakeObjTool{},
		UI:         &proftest.TestUI{T: t},
		HTTPServer: creator,
	}, false)
	<-serverCreated

	// Close the server when the test is done.
	t.Cleanup(server.Close)

	return server
}

func TestWebInterface(t *testing.T) {
	prof := makeFakeProfile()
	server := makeTestServer(t, prof)
	haveDot := false
	if _, err := exec.LookPath("dot"); err == nil {
		haveDot = true
	}

	type testCase struct {
		path    string
		want    []string
		needDot bool
	}
	testcases := []testCase{
		{"/", []string{"F1", "F2", "F3", "testbin", "cpu"}, true},
		{"/top", []string{`"Name":"F2","InlineLabel":"","Flat":200,"Cum":300,"FlatFormat":"200ms","CumFormat":"300ms"}`}, false},
		{"/source?f=" + url.QueryEscape("F[12]"), []string{
			"F1",
			"F2",
			`\. +300ms .*f1:asm`,    // Cumulative count for F1
			"200ms +300ms .*f2:asm", // Flat + cumulative count for F2
		}, false},
		{"/peek?f=" + url.QueryEscape("F[12]"),
			[]string{"300ms.*F1", "200ms.*300ms.*F2"}, false},
		{"/disasm?f=" + url.QueryEscape("F[12]"),
			[]string{"f1:asm", "f2:asm"}, false},
		{"/flamegraph", []string{
			"File: testbin",
			// Check profile frame JSON is included.
			`\\u0022n\\u0022:\\u0022root\\u0022`,
			`\\u0022n\\u0022:\\u0022F1\\u0022`,
			// Check d3-flame-graph JS is included.
			`flamegraph:\(\)=>|flamegraph.*function|function.*flamegraph`,
			// Check d3-flame-graph CSS is included.
			".d3-flame-graph rect {",
		}, false},
		{"/flamegraph2", []string{
			"File: testbin",
			// Check that interesting frames are included.
			`\bF1\b`,
			`\bF2\b`,
			// Check new view JS is included.
			`function stackViewer`,
			// Check new view CSS is included.
			"#stack-chart {",
		}, false},
	}
	for _, c := range testcases {
		if c.needDot && !haveDot {
			t.Log("skipping", c.path, "since dot (graphviz) does not seem to be installed")
			continue
		}
		res, err := http.Get(server.URL + c.path)
		if err != nil {
			t.Error("could not fetch", c.path, err)
			continue
		}
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Error("could not read response", c.path, err)
			continue
		}
		result := string(data)
		for _, w := range c.want {
			if match, _ := regexp.MatchString(w, result); !match {
				t.Errorf("response for %s does not match "+
					"expected pattern '%s'; "+
					"actual result:\n%s", c.path, w, result)
			}
		}
	}

	// Also fetch all the test case URLs in parallel to test thread
	// safety when run under the race detector.
	var wg sync.WaitGroup
	for _, c := range testcases {
		if c.needDot && !haveDot {
			continue
		}
		path := server.URL + c.path
		for count := 0; count < 2; count++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := http.Get(path)
				if err != nil {
					t.Error("could not fetch", c.path, err)
					return
				}
				if _, err = io.ReadAll(res.Body); err != nil {
					t.Error("could not read response", c.path, err)
				}
			}()
		}
	}
	wg.Wait()
}

// Implement fake object file support.

const addrBase = 0x1000
const fakeSource = "testdata/file1000.src"

type fakeObj struct{}

func (f fakeObj) Close() error                        { return nil }
func (f fakeObj) Name() string                        { return "testbin" }
func (f fakeObj) ObjAddr(addr uint64) (uint64, error) { return addr, nil }
func (f fakeObj) BuildID() string                     { return "" }
func (f fakeObj) SourceLine(addr uint64) ([]plugin.Frame, error) {
	return nil, fmt.Errorf("SourceLine unimplemented")
}
func (f fakeObj) Symbols(r *regexp.Regexp, addr uint64) ([]*plugin.Sym, error) {
	return []*plugin.Sym{
		{
			Name: []string{"F1"}, File: fakeSource,
			Start: addrBase, End: addrBase + 10,
		},
		{
			Name: []string{"F2"}, File: fakeSource,
			Start: addrBase + 10, End: addrBase + 20,
		},
		{
			Name: []string{"F3"}, File: fakeSource,
			Start: addrBase + 20, End: addrBase + 30,
		},
	}, nil
}

type fakeObjTool struct{}

func (obj fakeObjTool) Open(file string, start, limit, offset uint64, relocationSymbol string) (plugin.ObjFile, error) {
	return fakeObj{}, nil
}

func (obj fakeObjTool) Disasm(file string, start, end uint64, intelSyntax bool) ([]plugin.Inst, error) {
	return []plugin.Inst{
		{Addr: addrBase + 10, Text: "f1:asm", Function: "F1", Line: 3},
		{Addr: addrBase + 20, Text: "f2:asm", Function: "F2", Line: 11},
		{Addr: addrBase + 30, Text: "d3:asm", Function: "F3", Line: 22},
	}, nil
}

func makeFakeProfile() *profile.Profile {
	// Three functions: F1, F2, F3 with three lines, 11, 22, 33.
	funcs := []*profile.Function{
		{ID: 1, Name: "F1", Filename: fakeSource, StartLine: 3},
		{ID: 2, Name: "F2", Filename: fakeSource, StartLine: 5},
		{ID: 3, Name: "F3", Filename: fakeSource, StartLine: 7},
	}
	lines := []profile.Line{
		{Function: funcs[0], Line: 11},
		{Function: funcs[1], Line: 22},
		{Function: funcs[2], Line: 33},
	}
	mapping := []*profile.Mapping{
		{
			ID:             1,
			Start:          addrBase,
			Limit:          addrBase + 100,
			Offset:         0,
			File:           "testbin",
			HasFunctions:   true,
			HasFilenames:   true,
			HasLineNumbers: true,
		},
	}

	// Three interesting addresses: base+{10,20,30}
	locs := []*profile.Location{
		{ID: 1, Address: addrBase + 10, Line: lines[0:1], Mapping: mapping[0]},
		{ID: 2, Address: addrBase + 20, Line: lines[1:2], Mapping: mapping[0]},
		{ID: 3, Address: addrBase + 30, Line: lines[2:3], Mapping: mapping[0]},
	}

	// Two stack traces.
	return &profile.Profile{
		PeriodType:    &profile.ValueType{Type: "cpu", Unit: "milliseconds"},
		Period:        1,
		DurationNanos: 10e9,
		SampleType: []*profile.ValueType{
			{Type: "cpu", Unit: "milliseconds"},
		},
		Sample: []*profile.Sample{
			{
				Location: []*profile.Location{locs[2], locs[1], locs[0]},
				Value:    []int64{100},
			},
			{
				Location: []*profile.Location{locs[1], locs[0]},
				Value:    []int64{200},
			},
		},
		Location: locs,
		Function: funcs,
		Mapping:  mapping,
	}
}

func TestGetHostAndPort(t *testing.T) {
	if runtime.GOOS == "nacl" || runtime.GOOS == "js" {
		t.Skip("test assumes tcp available")
	}

	type testCase struct {
		hostport       string
		wantHost       string
		wantPort       int
		wantRandomPort bool
	}

	testCases := []testCase{
		{":", "localhost", 0, true},
		{":4681", "localhost", 4681, false},
		{"localhost:4681", "localhost", 4681, false},
	}
	for _, tc := range testCases {
		host, port, err := getHostAndPort(tc.hostport)
		if err != nil {
			t.Errorf("could not get host and port for %q: %v", tc.hostport, err)
		}
		if got, want := host, tc.wantHost; got != want {
			t.Errorf("for %s, got host %s, want %s", tc.hostport, got, want)
			continue
		}
		if !tc.wantRandomPort {
			if got, want := port, tc.wantPort; got != want {
				t.Errorf("for %s, got port %d, want %d", tc.hostport, got, want)
				continue
			}
		}
	}
}

func TestIsLocalHost(t *testing.T) {
	for _, s := range []string{"localhost:10000", "[::1]:10000", "127.0.0.1:10000"} {
		host, _, err := net.SplitHostPort(s)
		if err != nil {
			t.Error("unexpected error when splitting", s)
			continue
		}
		if !isLocalhost(host) {
			t.Errorf("host %s from %s not considered local", host, s)
		}
	}
}

func BenchmarkTop(b *testing.B)   { benchmarkURL(b, "/top", false) }
func BenchmarkFlame(b *testing.B) { benchmarkURL(b, "/flamegraph2", false) }
func BenchmarkDot(b *testing.B)   { benchmarkURL(b, "/", true) }

func benchmarkURL(b *testing.B, path string, needDot bool) {
	if needDot {
		if _, err := exec.LookPath("dot"); err != nil {
			b.Skip("dot not available")
		}
	}
	prof := largeProfile(b)
	server := makeTestServer(b, prof)
	url := server.URL + path
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := http.Get(url)
		if err != nil {
			b.Fatal(err)
		}
		data, err := io.ReadAll(res.Body)
		if err != nil {
			b.Fatal(err)
		}
		if i == 0 && testing.Verbose() {
			b.Logf("%-12s : %10d bytes", path, len(data))
		}
	}
}
