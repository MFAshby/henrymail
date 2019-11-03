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
	mxExpected := config.GetString(config.ServerName) + "."

	imapSrvTargetExpected, imapSrvPortExpected := config.GetString(config.ServerName)+".", config.GetString(config.ImapAddress)
	imapSrvTargetActual, imapSrvPortActualInteger := fetchSrvTargetAndPort("imap", "tcp")
	imapSrvPortActual := fmt.Sprintf(":%d", imapSrvPortActualInteger)
	imapSrvCorrect := imapSrvTargetExpected == imapSrvTargetActual && imapSrvPortExpected == imapSrvPortActual

	submissionSrvTargetExpected, submissionSrvPortExpected := config.GetString(config.ServerName)+".", config.GetString(config.MsaAddress)
	submissionSrvTargetActual, submissionSrvPortActualInteger := fetchSrvTargetAndPort("submission", "tcp")
	submissionSrvPortActual := fmt.Sprintf(":%d", submissionSrvPortActualInteger)
	submissionSrvCorrect := submissionSrvTargetExpected == submissionSrvTargetActual && submissionSrvPortExpected == submissionSrvPortActual

	data := struct {
		layoutData
		DkimRecordIs                string
		DkimRecordShouldBe          string
		SpfRecordIs                 string
		SpfRecordShouldBe           string
		MxRecordIs                  string
		MxRecordShouldBe            string
		ImapSrvCorrect              bool
		ImapSrvTargetShouldBe       string
		ImapSrvPortShouldBe         string
		SubmissionSrvCorrect        bool
		SubmissionSrvTargetShouldBe string
		SubmissionSrvPortShouldBe   string
		FailingPorts                string
	}{
		layoutData:                  *ld,
		DkimRecordIs:                dkimActual,
		DkimRecordShouldBe:          dkimExpected,
		SpfRecordIs:                 spfActual,
		SpfRecordShouldBe:           spfExpected,
		MxRecordIs:                  mxActual,
		MxRecordShouldBe:            mxExpected,
		ImapSrvCorrect:              imapSrvCorrect,
		ImapSrvTargetShouldBe:       imapSrvTargetExpected,
		ImapSrvPortShouldBe:         imapSrvPortExpected,
		SubmissionSrvCorrect:        submissionSrvCorrect,
		SubmissionSrvTargetShouldBe: submissionSrvTargetExpected,
		SubmissionSrvPortShouldBe:   submissionSrvPortExpected,
		FailingPorts:                fetchFailingPorts(),
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

func fetchSrvTargetAndPort(service, proto string) (string, int) {
	cname, addrs, err := net.LookupSRV(service, proto, config.GetString(config.Domain))
	srvActual := ""
	srvPort := 0
	if err != nil {
		srvActual = err.Error()
	} else {
		srvLen := len(addrs)
		if srvLen != 1 {
			srvActual = fmt.Sprintf("Expecting exactly 1 srv record, found %d", srvLen)
		} else {
			srvActual = addrs[0].Target
			srvPort = int(addrs[0].Port)
		}
	}
	print(cname)
	return srvActual, srvPort
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
