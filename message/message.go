package message

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/paulrosania/go-mail"
)

const (
	HeaderFrom       = mail.FromFieldName
	HeaderMessageId  = mail.MessageIDFieldName
	HeaderNewsgroups = "Newsgroups"
)

type Message struct {
	*mail.Message
}

func Read(r io.Reader) (*Message, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ReadBytes(data)
}

func ReadBytes(data []byte) (*Message, error) {
	msg, err := mail.ReadMessage(string(data))
	if err != nil {
		return nil, err
	}
	return &Message{msg}, nil
}

func (m *Message) Addresses(header string) ([]string, []string) {
	var resAddrs, resNames []string
	for _, addr := range m.Header.Addresses(header) {
		resNames = append(resNames, addr.Name(false))
		resAddrs = append(resAddrs, AddressString(addr))
	}
	return resAddrs, resNames
}

func (m *Message) HeaderValues(header string) []string {
	var res []string
	for _, field := range m.Header.Fields {
		if strings.ToLower(field.Name()) == strings.ToLower(header) {
			res = append(res, field.Value())
		}
	}
	return res
}

func (m *Message) HeaderValue(header string) string {
	return strings.Join(m.HeaderValues(header), " ")
}

func (m *Message) Size() (bytes int, lines int) {
	return len([]byte(m.Data)), strings.Count(m.Data, "\n")
}

/*
func (m *Message) PGPSignature() bool {
}
*/
