package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

func (m *Mailer) connect(ctx context.Context) (c *client.Client, err error) {
	log.Printf("INFO: Connecting to IMAP server %s...", m.Host)

	// Connect to server
	c, err = client.Dial(m.Host + ":" + m.ImapPort)
	if err != nil {
		return
	}

	if m.ImapDebug {
		c.SetDebug(os.Stderr)
	}

	// Don't forget to logout
	defer func() {
		if c != nil && err != nil {
			t, cancel := context.WithTimeout(context.Background(), time.Second)
			go func() {
				err := c.Logout()
				if err != nil {
					log.Printf("ERROR: %v", err)
				}
				cancel()
			}()
			<-t.Done()
			err = c.Close()
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
			c = nil
		}
	}()

	// Start a TLS session
	err = c.StartTLS(&tls.Config{ServerName: m.Host})
	if err != nil {
		return
	}
	log.Println("INFO: TLS established")

	// Login
	err = c.Login(m.User, m.Pass)
	if err != nil {
		return
	}
	log.Printf("INFO: Logged in as %s", m.User)

	// List mailboxes
	//mailboxes := make(chan *imap.MailboxInfo, 10)
	//done := make(chan error, 1)
	//go func() {
	//	done <- c.List("", "*", mailboxes)
	//}()

	//log.Println("Mailboxes:")
	//for m := range mailboxes {
	//	log.Printf("* %s = %+v", m.Name, m)
	//}

	//err = <-done
	//if err != nil {
	//	return
	//}

	// Select INBOX
	var mbox *imap.MailboxStatus
	mbox, err = c.Select("INBOX", false)
	if err != nil {
		return
	}

	err = m.readMessages(ctx, c, mbox)
	return
}

func (m *Mailer) clientLoop(ctx context.Context, wg *sync.WaitGroup, c *client.Client) {
	var err error
	defer func() {
		wg.Done()
		log.Print("INFO: Stopped IMAP client")
	}()

	for ctx.Err() == nil {
		if c != nil {
			err = m.run(ctx, c)
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}

		if ctx.Err() != nil {
			break
		}

		log.Print("INFO: Reconnecting IMAP...")
		c, err = m.connect(ctx)
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
	}
}

func (m *Mailer) readMessages(ctx context.Context, c *client.Client, mbox *imap.MailboxStatus) error {
	if mbox.Messages == 0 {
		return nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(1, mbox.Messages)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	section := &imap.BodySectionName{} // whole message
	log.Printf("DEBUG: read messages from %d to %d", 1, mbox.Messages)
	go func() {
		err := c.Fetch(seqset, []imap.FetchItem{section.FetchItem()}, messages)
		done <- err
	}()

loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			if err != nil {
				return err
			}
			break loop
		case msg := <-messages:
			if msg == nil {
				break loop
			}
			log.Printf("INFO: IMAP received message")
			err := m.readMessage(msg)
			if err != nil {
				return err
			}

			// First mark the message as deleted
			delseqset := new(imap.SeqSet)
			delseqset.AddNum(msg.SeqNum)
			item := imap.FormatFlagsOp(imap.AddFlags, true)
			flags := []interface{}{imap.DeletedFlag}
			if err := c.Store(delseqset, item, flags, nil); err != nil {
				return err
			}
		}
	}

	log.Printf("DEBUG: reading finished, expunge messages...")

	return c.Expunge(nil)
}

func (m *Mailer) readMessage(msg *imap.Message) error {
	section := &imap.BodySectionName{} // whole message
	part := msg.GetBody(section)
	if part == nil {
		return fmt.Errorf("No part available")
	}
	r, err := mail.CreateReader(part)
	if err != nil {
		return err
	}

	for part, err := r.NextPart(); err != io.EOF; part, err = r.NextPart() {
		var data []byte
		data, err = ioutil.ReadAll(part.Body)
		if err != nil {
			return err
		}
		mat := validationUuidRegexp.FindStringSubmatch(string(data))
		if mat == nil {
			continue
		}
		tok := mat[1]
		mat = validationTokenRegexp(tok).FindStringSubmatch(string(data))
		if mat == nil {
			continue
		}
		token := mat[1]
		mat = validationEmailRegexp(tok).FindStringSubmatch(string(data))
		if mat == nil {
			continue
		}
		email := mat[1]
		log.Printf("INFO: IMAP received validation for %s with token %s", email, token)
		err := m.Validations.ReceivedEmailToken(email, token)
		if err != nil {
			return err
		}
	}
	return nil
}

var validationUuidRegexp = regexp.MustCompile("(\\S*):" + regexp.QuoteMeta(UuidEmailValidation))

func validationTokenRegexp(uniqueTok string) *regexp.Regexp {
	return regexp.MustCompile(regexp.QuoteMeta(uniqueTok) + ":t:(\\S*)")
}
func validationEmailRegexp(uniqueTok string) *regexp.Regexp {
	return regexp.MustCompile(regexp.QuoteMeta(uniqueTok) + ":e:(\\S*)")
}

func (m *Mailer) run(ctx context.Context, c *client.Client) error {
	defer func() {
		log.Print("INFO: Disconnecting IMAP...")
		t, cancel := context.WithTimeout(context.Background(), time.Second)
		go func() {
			err := c.Logout()
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
			cancel()
		}()
		<-t.Done()
		err := c.Close()
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
		log.Print("INFO: Disconnected IMAP...")
	}()

	for ctx.Err() == nil {
		mbox, err := m.idle(ctx, c)
		if err != nil {
			return err
		}

		if mbox != nil {
			err := m.readMessages(ctx, c, mbox)
			if err != nil {
				return err
			}
		}
	}

	return ctx.Err()
}

func (m *Mailer) idle(ctx context.Context, c *client.Client) (mbox *imap.MailboxStatus, err error) {
	idleClient := idle.NewClient(c)

	// Create a channel to receive mailbox updates
	updates := make(chan client.Update)
	c.Updates = updates
	defer func() { c.Updates = nil }()

	// Start idling
	idleCtx, idleCancel := context.WithCancel(ctx)
	defer idleCancel()
	done := make(chan error, 1)
	go func() {
		log.Print("INFO: IMAP idling...")
		err := idleClient.IdleWithFallback(idleCtx.Done(), 0)
		log.Printf("DEBUG: IMAP idle err: %v", err)
		done <- err
	}()

	for {
		select {
		case <-ctx.Done():
			return mbox, ctx.Err()
		case update := <-updates:
			switch upd := update.(type) {
			case *client.MailboxUpdate:
				mbox = upd.Mailbox
				log.Printf("INFO: IMAP mailbox update, %d messages", mbox.Messages)
				idleCancel()
			default:
				//log.Printf("DEBUG: New update: %t %+s", update, update)
			}
		case err := <-done:
			log.Println("DEBUG: Not idling anymore")
			return mbox, err
		}
	}
}
