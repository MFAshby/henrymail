package web

import (
	"fmt"
	"henrymail/config"
	"henrymail/dkim"
	"henrymail/models"
	"henrymail/spf"
	"net"
	"net/http"
	"strings"
)

func (wa *wa) healthChecks(w http.ResponseWriter, r *http.Request, u *models.User) {
	ld, e := wa.layoutData(u)
	if e != nil {
		wa.renderError(w, e)
		return
	}

	spfExpected := spf.GetSpfRecordString()
	spfActual := fetchSpfActual()

	dkimExpected, e := dkim.GetDkimRecordString(wa.db)
	if e != nil {
		wa.renderError(w, e)
		return
	}
	dkimActual := fetchDkimActual()
	mxActual := fetchMxActual()
	mxExpected := config.GetString(config.ServerName) + "." // MX records expect a trailing dot

	data := struct {
		layoutData
		DkimRecordIs       string
		DkimRecordShouldBe string
		SpfRecordIs        string
		SpfRecordShouldBe  string
		MxRecordIs         string
		MxRecordShouldBe   string
		FailingPorts       string
	}{
		layoutData:         *ld,
		DkimRecordIs:       dkimActual,
		DkimRecordShouldBe: dkimExpected,
		SpfRecordIs:        spfActual,
		SpfRecordShouldBe:  spfExpected,
		MxRecordIs:         mxActual,
		MxRecordShouldBe:   mxExpected,
		FailingPorts:       fetchFailingPorts(),
	}
	wa.healthChecksView.render(w, data)
}

func fetchDkimActual() string {
	dkimActual := ""
	dkimRecords, e := net.LookupTXT("mx._domainkey." + config.GetString(config.Domain))
	if e != nil {
		dkimActual = e.Error()
	} else {
		// Longer TXT records may be split
		dkimActual = strings.Join(dkimRecords, "")
	}
	return dkimActual
}

func fetchSpfActual() string {
	spfActual := ""
	spfRecords, e := net.LookupTXT(config.GetString(config.Domain))
	if e != nil {
		spfActual = e.Error()
	} else {
		for _, s := range spfRecords {
			if strings.Contains(s, "v=spf1") {
				spfActual = s
			}
		}
	}
	return spfActual
}

func fetchMxActual() string {
	mxActual := ""
	mxes, e := net.LookupMX(config.GetString(config.Domain))
	if e != nil {
		mxActual = e.Error()
	} else {
		mxesLen := len(mxes)
		if mxesLen != 1 {
			// Should be just 1 MX record
			mxActual = fmt.Sprintf("Expecting exactly 1 mx record, found %d", mxesLen)
		} else {
			// MX record should be referring to this server
			mxActual = mxes[0].Host
		}
	}
	return mxActual
}

func fetchFailingPorts() string {
	// Should be able to open a socket on these ports, to ourselves
	portsTest := []string{
		"25",
		"143",
		"443",
		"587",
	}
	var failedPorts []string
	for _, port := range portsTest {
		con, e := net.Dial("tcp", config.GetString(config.ServerName)+":"+port)
		if e != nil {
			failedPorts = append(failedPorts, port)
		} else {
			_ = con.Close()
		}
	}
	return strings.Join(failedPorts, ",")
}
