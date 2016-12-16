package main

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

var (
	mux        *http.ServeMux
	dnsserver  *PuppetDnsServer
	httpserver *httptest.Server
)

func setup() {
	mux = http.NewServeMux()
	httpserver = httptest.NewServer(mux)

	serverUrl, _ := url.Parse(httpserver.URL)
	dnsserver = &PuppetDnsServer{
		"puppet.be",
		"12345",
		"127.0.0.1",
		serverUrl.String(),
		600,
		true,
		[][]string{[]string{"role", "hostgroup"}, []string{"role", "os"}},
	}
}

func teardown() {
	httpserver.Close()
}

func TestBuildPuppetDbQuery(t *testing.T) {
	cases := []struct {
		fact_names []string
		values     []string
		result     string
	}{
		{[]string{"os", "kernel"}, []string{"Linux", "4.8.0"},
			`["and", ["=", ["fact", "os"], "Linux"], ["=", ["fact", "kernel"], "4.8.0"]]`,
		},
		{[]string{"os"}, []string{"Linux"},
			`["and", ["=", ["fact", "os"], "Linux"]]`,
		},
	}
	for _, c := range cases {
		query := buildPuppetDbQuery(c.fact_names, c.values)
		if query != c.result {
			t.Errorf("buildPuppetDbQuery(%q, %q) == %q, want %q", c.fact_names, c.values, query, c.result)
		}
	}

}

func RunLocalUDPServer(laddr string) (*dns.Server, string, error) {
	server, l, _, err := RunLocalUDPServerWithFinChan(laddr)

	return server, l, err
}

func RunLocalUDPServerWithFinChan(laddr string) (*dns.Server, string, chan struct{}, error) {
	pc, err := net.ListenPacket("udp", laddr)
	if err != nil {
		return nil, "", nil, err
	}
	server := &dns.Server{PacketConn: pc, ReadTimeout: time.Hour, WriteTimeout: time.Hour}

	waitLock := sync.Mutex{}
	waitLock.Lock()
	server.NotifyStartedFunc = waitLock.Unlock

	fin := make(chan struct{}, 0)

	go func() {
		server.ActivateAndServe()
		close(fin)
		pc.Close()
	}()

	waitLock.Lock()
	return server, pc.LocalAddr().String(), fin, nil
}

func TestQueryNodes(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v3/nodes",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `[{"name":"affinitic.be",
							 "dactivated":null,
							 "catalog_timestamp" : "2014-01-10T21:17:03.467Z",
							 "facts_timestamp" : "2014-01-10T21:15:40.933Z",
							 "report_timestamp" : "2014-01-10T21:17:30.877Z" }]`)
		})

	dns.HandleFunc("puppet.be.", dnsserver.handleRequest)
	defer dns.HandleRemove("puppet.be.")
	m1 := new(dns.Msg)
	m1.SetQuestion("webserver.linux.puppet.be.", dns.TypeA)

	s, addrstr, err := RunLocalUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("unable to run test server: %s", err)
	}
	defer s.Shutdown()

	c := new(dns.Client)
	r, _, err := c.Exchange(m1, addrstr)
	if err != nil {
		t.Errorf("failed to exchange: %v", err)
	}
	if r != nil && r.Rcode != dns.RcodeSuccess {
		t.Errorf("failed to get an valid answer\n%v", r)
	}
	if len(r.Answer) != 1 {
		t.Errorf("answer has wrong size\n%v", r)
	}
	if r, ok := r.Answer[0].(*dns.A); ok {
		if r.A.String() != "91.121.100.12" {
			t.Errorf("wrong dns response", r.A)
		}
		if r.Header().Ttl != 600 {
			t.Errorf("wrong ttl", r.Header().Ttl)
		}
	}

}

func TestMatchFirstItemInHierarychy(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/v3/nodes",
		func(w http.ResponseWriter, r *http.Request) {
			puppetdb_query := r.URL.Query()["query"][0]
			if puppetdb_query == `["and", ["=", ["fact", "role"], "webserver"], ["=", ["fact", "hostgroup"], "linux"]]` {
				fmt.Fprint(w, `[{"name":"affinitic.be",
							 "dactivated":null,
							 "catalog_timestamp" : "2014-01-10T21:17:03.467Z",
							 "facts_timestamp" : "2014-01-10T21:15:40.933Z",
							 "report_timestamp" : "2014-01-10T21:17:30.877Z" }]`)
			} else {
				fmt.Fprint(w, "")
			}
		})

	dns.HandleFunc("puppet.be.", dnsserver.handleRequest)
	defer dns.HandleRemove("puppet.be.")
	m1 := new(dns.Msg)
	m1.SetQuestion("webserver.linux.puppet.be.", dns.TypeA)

	s, addrstr, err := RunLocalUDPServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("unable to run test server: %s", err)
	}
	defer s.Shutdown()

	c := new(dns.Client)
	r, _, err := c.Exchange(m1, addrstr)
	if err != nil {
		t.Errorf("failed to exchange: %v", err)
	}
	if r != nil && r.Rcode != dns.RcodeSuccess {
		t.Errorf("failed to get an valid answer\n%v", r)
	}
	if len(r.Answer) != 1 {
		t.Errorf("answer has wrong size\n%v", r)
	}
	if r, ok := r.Answer[0].(*dns.A); ok {
		if r.A.String() != "91.121.100.12" {
			t.Errorf("wrong dns response", r.A)
		}
		if r.Header().Ttl != 600 {
			t.Errorf("wrong ttl", r.Header().Ttl)
		}
	}
}
