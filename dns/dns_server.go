package dns

import (
	"database/sql"
	"fmt"
	"github.com/miekg/dns"
	"henrymail/config"
	"henrymail/dkim"
	"henrymail/spf"
	"log"
	"net"
	"strings"
)

func StartFakeDNS(db *sql.DB, addr, proto string) {
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
					result, _ = dkim.GetDkimRecordString(db)
				} else {
					result = spf.GetSpfRecordString()
				}
				m.Answer = append(m.Answer, &dns.TXT{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: dns.ClassINET, Ttl: 0},
					Txt: chunk(result, 255),
				})
			case dns.TypeSRV:
				target := ""
				port := uint16(0)
				if strings.Contains(q.Name, "_imap._tcp") {
					target = config.GetString(config.ServerName) + "."
					_, _ = fmt.Sscanf(config.GetString(config.ImapAddress), ":%d", &port)
				} else if strings.Contains(q.Name, "_submission._tcp") {
					target = config.GetString(config.ServerName) + "."
					_, _ = fmt.Sscanf(config.GetString(config.MsaAddress), ":%d", &port)
				}

				if target != "" {
					m.Answer = append(m.Answer, &dns.SRV{
						Hdr:      dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: dns.ClassINET, Ttl: 0},
						Target:   target,
						Port:     port,
						Priority: uint16(10),
						Weight:   uint16(5),
					})
				}
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
		chunks = append(chunks, buf[:])
	}
	return chunks
}
