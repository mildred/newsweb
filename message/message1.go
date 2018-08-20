package message

import (
	"bytes"
	"io"
	//"net/textproto"

	"github.com/emersion/go-message/mail"
)

type Message1 struct {
	Reader *mail.Reader
}

func Read1(data io.Reader) (*Message1, error) {
	r, err := mail.CreateReader(data)
	if err != nil {
		return nil, err
	}
	return &Message1{r}, nil
}

func ReadBytes1(data []byte) (*Message1, error) {
	return Read1(bytes.NewReader(data))
}

func (m *Message1) Addresses(header string) ([]string, []string, error) {
	addrs, err := m.Reader.Header.AddressList(header)
	if err != nil {
		return nil, nil, err
	}
	var resAddrs, resNames []string
	for _, addr := range addrs {
		resAddrs = append(resAddrs, addr.Address)
		resNames = append(resNames, addr.Name)
	}
	return resAddrs, resNames, nil
}

/*
func (m *Message1) HeaderValues(header string) []string {
}

func (m *Message) HeaderValue(header string) string {
}

func (m *Message) Size() (bytes int, lines int) {
}

func (m *Message) TextprotoHeaders() textproto.MIMEHeader {
}

func (m *Message) PGPSignature() bool {
}
*/
