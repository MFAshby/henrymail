package main

import (
	"github.com/miekg/dns"
	"henrymail/config"
	"henrymail/dkim"
	"henrymail/spf"
	"log"
	"net"
	"strings"
)

/**
 * Fake DNS server for local testing.
 * This lets us pretend that our DKIM and TXT records are right
 */
func StartFakeDns(addr, proto string) {
	s := &dns.Server{
		Addr: addr,
		Net:  proto,
	}
	dns.HandleFunc(config.GetString(config.Domain)+".", func(writer dns.ResponseWriter, r *dns.Msg) {
		//log.Printf("DNS Request %v", r)
		m := new(dns.Msg)
		m.SetReply(r)
		for _, q := range r.Question {
			switch q.Qtype {
			case dns.TypeA:
				m.Answer = append(m.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: dns.ClassINET, Ttl: 0},
					A:   net.IPv4(127, 0, 0, 1),
				})
			case dns.TypeMX:
				m.Answer = append(m.Answer, &dns.MX{
					Hdr:        dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: dns.ClassINET, Ttl: 0},
					Preference: 10,
					Mx:         config.GetString(config.ServerName) + ".",
				})
			case dns.TypeTXT:
				result := ""
				if strings.Contains(q.Name, "mx._domainkey.") {
					result, _ = dkim.GetDkimRecordString()
				} else {
					result = spf.GetSpfRecordString()
				}
				m.Answer = append(m.Answer, &dns.TXT{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: dns.ClassINET, Ttl: 0},
					Txt: chunk(result, 255),
				})
			}
		}

		//log.Printf("Reply %v", m)
		e := writer.WriteMsg(m)
		if e != nil {
			log.Print(e)
		}
	})

	go func() { log.Fatal(s.ListenAndServe()) }()
	log.Printf("Started FAKE DNS SERVER at " + config.GetString(config.FakeDnsAddress))
}

func chunk(buf string, lim int) []string {
	var chunk string
	chunks := make([]string, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks
}
