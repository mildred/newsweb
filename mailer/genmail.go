package mailer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/paulrosania/go-mail"
)

const (
	UuidEmailValidation = "8ce7db75-31c1-4308-974e-0971c19fa158"
)

func genHexToken(size int) string {
	var data = make([]byte, size)
	_, _ = rand.Read(data)
	var enc bytes.Buffer
	base64.NewEncoder(base64.RawURLEncoding, &enc).Write(data)
	return string(enc.Bytes())
}

func (m *Mailer) GenValidationMail(to, token string) []byte {
	tok := genHexToken(16)
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

mail type:      %s:%s
secret token:   %s:t:%s
e-mail address: %s:e:%s
------------------------------------------------------------
`, to, tok, UuidEmailValidation, tok, token, tok, to),
		},
	}

	return []byte(msg.RFC822(false))
}
