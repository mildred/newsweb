package message

import (
	"bytes"

	"github.com/paulrosania/go-mail"
)

// AddressLocalpartIsSensible is taken from
// github.com/paulrosania/go-mail/addresses.go
// Returns true if this is a sensible-looking localpart, and false if it needs
// quoting. We should never permit one of our users to need quoting, but we
// must permit foreign addresses that do.
func AddressLocalpartIsSensible(localPart string) bool {
	if localPart == "" {
		return false
	}
	i := 0
	for i < len(localPart) {
		c := localPart[i]
		if c == '.' {
			if localPart[i+1] == '.' {
				return false
			}
		} else if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '!' || c == '#' ||
			c == '$' || c == '%' ||
			c == '&' || c == '\'' ||
			c == '*' || c == '+' ||
			c == '-' || c == '/' ||
			c == '=' || c == '?' ||
			c == '^' || c == '_' ||
			c == '`' || c == '{' ||
			c == '|' || c == '}' ||
			c == '~' || c >= 161) {
			return false
		}
		i++
	}
	return true
}

// QuoteLocalPart is taken from github.com/paulrosania/go-mail/strings.go
// Returns a version of this string quited with \a c, and where any occurences
// of \a c or \a q are escaped with \a q.
func QuoteLocalPart(str string, c, q byte) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(str)+2))
	buf.WriteByte(c)
	i := 0
	for i < len(str) {
		if str[i] == c || str[i] == q {
			buf.WriteByte(q)
		}
		buf.WriteByte(str[i])
		i++
	}
	buf.WriteByte(c)
	return buf.String()
}

func AddressString(addr mail.Address) string {
	if addr.Localpart == "" && addr.Domain == "" {
		return ""
	}
	localPart := addr.Localpart
	if !AddressLocalpartIsSensible(localPart) {
		localPart = QuoteLocalPart(localPart, '"', '\'')
	}
	if addr.Domain == "" {
		return localPart
	} else {
		return localPart + "@" + addr.Domain
	}
}
