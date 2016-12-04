package main

import (
	"flag"
	"fmt"
	"github.com/akira/go-puppetdb"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var ipv4, conf string
var ttl int
var logflag bool
var mapv4 map[string]string

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type PuppetDnsServer struct {
	Domain    string
	Port      string
	Bind      string
	Puppetdb  string
	Verbose   bool
	Hierarchy [][]string
}

func parseConfig(conf string) *PuppetDnsServer {
	p := PuppetDnsServer{}
	data, err := ioutil.ReadFile(conf)
	check(err)
	err = yaml.Unmarshal(data, &p)
	if p.Verbose {
		fmt.Printf("Domain: %s\n", p.Domain)
	}
	check(err)
	return &p
}

func (p PuppetDnsServer) start() {
	b := net.JoinHostPort(p.Bind, p.Port)
	if p.Verbose {
		fmt.Printf("Listening on %s\n", b)
	}
	dns.HandleFunc(".", p.handleRequest)
	go func() {
		srv := &dns.Server{Addr: b, Net: "udp"}
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal("Failed to set udp listener %s\n", err.Error())
		}
	}()
	go func() {
		srv := &dns.Server{Addr: b, Net: "tcp"}
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal("Failed to set tcp listener %s\n", err.Error())
		}
	}()
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case s := <-sig:
			log.Fatalf("Signal (%d) received, stopping\n", s)
		}
	}

}

func main() {
	flag.StringVar(&conf, "conf", "dns.conf", "Config File")
	flag.Parse()
	server := parseConfig(conf)
	server.start()
}

func buildPuppetDbQuery(facts []string, args []string) string {
	queryParts := []string{}
	queryParts = append(queryParts, `"and"`)
	for i, fact := range facts {
		queryParts = append(queryParts, fmt.Sprintf(`["=", ["fact", "%s"], "%s"]`, fact, args[i]))
	}
	return fmt.Sprintf("[%s]", strings.Join(queryParts, ", "))
}

func (p PuppetDnsServer) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	domain := r.Question[0].Name
	if p.Verbose {
		ip, _, _ := net.SplitHostPort(w.RemoteAddr().String())
		fmt.Printf("Received DNS query: %s\t%s\n", ip, domain)
	}
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	if strings.HasSuffix(domain, fmt.Sprintf("%s.", p.Domain)) {
		client := puppetdb.NewClient(p.Puppetdb, true)
		ret := []puppetdb.NodeJson{}
		query := map[string]string{}
		subdomain := strings.Replace(domain, p.Domain, "", -1)
		args := strings.Split(subdomain, ".")
		args = args[:len(args)-2]
		for _, facts := range p.Hierarchy {
			if len(facts) == len(args) && len(m.Answer) == 0 {
				query["query"] = buildPuppetDbQuery(facts, args)
				if p.Verbose {
					fmt.Printf("Sending puppetdb query: %s\n", query["query"])
				}
				client.Get(&ret, "nodes", query)
				for _, node := range ret {
					ips, _ := net.LookupHost(node.Name)
					for _, ip := range ips {
						rr1 := new(dns.A)
						rr1.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(ttl)}
						rr1.A = net.ParseIP(ip)
						m.Answer = append(m.Answer, dns.RR(rr1))
						if p.Verbose {
							fmt.Printf("dns answer: %s\n", ip)
						}
					}
				}

			}
		}
	}
	w.WriteMsg(m)
}
