package mailer

import (
	"context"
	"io/ioutil"
	"net/smtp"
	"os"
	"strings"
	"sync"
)

type Mailer struct {
	Host        string
	SmtpPort    string
	ImapPort    string
	Mail        string
	User        string
	Pass        string
	PassFd      int
	PassFile    string
	ImapDebug   bool
	Validations Validations
}

type Validations interface {
	ReceivedEmailToken(email, token string) error
}

func (m *Mailer) Start(ctx context.Context, wg *sync.WaitGroup) error {
	var f *os.File
	if m.User == "" {
		m.User = m.Mail
	}

	if m.PassFd > 0 {
		f = os.NewFile(uintptr(m.PassFd), "smtp-pass-fd")
	} else if m.PassFile != "" {
		var err error
		f, err = os.Open(m.PassFile)
		if err != nil {
			return err
		}
	}

	if f != nil {
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		m.Pass = strings.TrimSpace(string(data))
	}

	c, err := m.connect(ctx)
	if err != nil {
		return err
	}

	wg.Add(1)
	go m.clientLoop(ctx, wg, c)

	return nil
}

func (m *Mailer) Send(mail []byte, to ...string) error {
	auth := smtp.PlainAuth("", m.User, m.Pass, m.Host)
	return smtp.SendMail(m.Host+":"+m.SmtpPort, auth, m.Mail, to, mail)
}
