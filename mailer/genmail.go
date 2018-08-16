package mailer

import (
	"fmt"
	"time"

	"github.com/paulrosania/go-mail"
)

func (m *Mailer) GenValidationMail(to, token string) []byte {
	msg := mail.NewMessage()
	msg.Header = &mail.Header{}
	msg.Header.Add("From", m.Mail)
	msg.Header.Add("To", to)
	msg.Header.Add("Date", time.Now().Format(time.RFC822))
	msg.Header.Add("Subject", "Please confirm your e-mail address")
	msg.Header.Add("Content-Type", "text/plain")
	msg.Parts = []*mail.Part{
		&mail.Part{
			Text: fmt.Sprintf(`Please confirm your e-mail address

You, or someone that pass for you, is trying to send a message using the e-mail
address: %s

If you are not the author of the message, you can ignore this e-mail and the
original message will be ignored.

If you are the author of the message, you need to reply to this message to
confirm you are the sender. If not, the message is going to be discarded. When
replying, you need to sign the message using your PGP secret key.


------------------------------------------------------------
Please keep the following text in your reply:

secret token:   %s
e-mail address: %s
------------------------------------------------------------
`, to, token, to),
		},
	}

	return []byte(msg.RFC822(false))
}
