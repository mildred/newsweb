package server

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/textproto"
	"strings"

	"github.com/dustin/go-nntp"
	"github.com/dustin/go-nntp/server"
	"github.com/paulrosania/go-mail"

	"github.com/mildred/newsweb/articles"
	"github.com/mildred/newsweb/message"
)

func convertGroup(grp *articles.Group) *nntp.Group {
	return &nntp.Group{
		Name:        grp.Name,
		Description: grp.Description,
		Count:       grp.Count,
		High:        grp.High,
		Low:         grp.Low,
		Posting:     nntp.PostingPermitted,
	}
}

type Connection struct {
	Server *Server
}

func (s *Connection) ListGroups(max int) (res []*nntp.Group, err error) {
	grps, err := s.Server.Articles.ListGroups()
	if err != nil {
		log.Printf("FATAL: %v", err)
		return nil, nntpserver.ErrFault
	}
	for _, grp := range grps {
		if max >= 0 && len(res) >= max {
			log.Printf("DEBUG: max groups of %d reached", max)
			break
		}
		log.Printf("DEBUG: list group %s", grp.Name)
		res = append(res, convertGroup(grp))
	}
	return res, nil
}

func (s *Connection) GetGroup(name string) (*nntp.Group, error) {
	grp, err := s.Server.Articles.GetGroup(name)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, nntpserver.ErrNoSuchGroup
	}
	return convertGroup(grp), nil
}

func (s *Connection) GetArticleNum(group *nntp.Group, num int64) (io.ReadCloser, string, error) {
	art, msgId, err := s.Server.Articles.GetArticleNum(group.Name, num)
	if err == articles.ErrNoGroup {
		return nil, "0", nntpserver.ErrNoSuchGroup
	} else if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, "0", nntpserver.ErrFault
	} else if art == nil {
		return nil, "0", nntpserver.ErrInvalidArticleNumber
	}

	if msgId == "" {
		msgId = "0"
	}

	return art, msgId, nil
}

func (s *Connection) GetArticleMsgId(group *nntp.Group, id string) (io.ReadCloser, int64, error) {
	art, num, err := s.Server.Articles.GetArticleMsgId(group.Name, id)
	if err == articles.ErrNoGroup {
		return nil, -1, nntpserver.ErrNoSuchGroup
	} else if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, -1, nntpserver.ErrFault
	} else if art == nil {
		return nil, -1, nntpserver.ErrInvalidMessageID
	}

	return art, num, nil
}

func (s *Connection) GetArticles(group *nntp.Group, from, to int64) ([]nntpserver.NumberedArticle, error) {
	var res []nntpserver.NumberedArticle
	for num := from; num <= to; num++ {
		art, _, err := s.Server.Articles.GetArticleNum(group.Name, num)
		if err == articles.ErrNoGroup {
			return nil, nntpserver.ErrNoSuchGroup
		} else if err != nil {
			log.Printf("ERROR: %v", err)
			return nil, nntpserver.ErrFault
		}
		if art == nil {
			continue
		}
		defer art.Close()

		artBytes, err := ioutil.ReadAll(art)
		if err != nil {
			log.Printf("ERROR: %v", err)
			return nil, nntpserver.ErrFault
		}

		msg, err := mail.ReadMessage(string(artBytes))
		if err != nil {
			log.Printf("ERROR: %v", err)
			return nil, nntpserver.ErrFault
		}

		a := nntpserver.NumberedArticle{
			Num: num,
			Article: &nntp.Article{
				Body:   nil,
				Bytes:  msg.RFC822Size,
				Lines:  bytes.Count(artBytes, []byte("\n")),
				Header: textproto.MIMEHeader{},
			},
		}
		for _, header := range msg.Header.Fields {
			a.Article.Header.Add(header.Name(), header.Value())
		}
		res = append(res, a)
	}
	return res, nil
}

func (s *Connection) Authorized() bool {
	return true
}

// Authenticate and optionally swap out the backend for this session.
// You may return nil to continue using the same backend.
func (s *Connection) Authenticate(user, pass string) (nntpserver.Backend, error) {
	return nil, nil
}

func (s *Connection) AllowPost() bool {
	return true
}

func (s *Connection) Post(article io.Reader) error {
	var buffer = new(bytes.Buffer)
	_, err := io.Copy(buffer, article)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nntpserver.ErrPostingFailed
	}

	msg, err := mail.ReadMessage(string(buffer.Bytes()))
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nntpserver.ErrPostingFailed
	}

	from := msg.Header.Addresses(mail.FromFieldName)
	fromAddr := message.AddressString(from[0])
	token, err := s.Server.Validations.GenValidationToken(fromAddr)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nntpserver.ErrPostingFailed
	}

	validationMail := s.Server.Mailer.GenValidationMail(fromAddr, token)
	err = s.Server.Mailer.Send(validationMail, fromAddr)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nntpserver.ErrPostingFailed
	}

	var msgId string
	var groups []string
	for _, hdr := range msg.Header.Fields {
		if strings.ToLower(hdr.Name()) == "newsgroups" {
			groups = append(groups, hdr.Value())
		} else if strings.ToLower(hdr.Name()) == "message-id" {
			if msgId != "" {
				log.Print("ERROR: Duplicate Message-Id")
				return nntpserver.ErrPostingFailed
			}
			msgId = hdr.Value()
		}
	}
	if len(groups) == 0 {
		log.Print("ERROR: Newsgroup header absent")
		return nntpserver.ErrPostingFailed
	}

	err = s.Server.Articles.Post(groups, msgId, buffer.Bytes())
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nntpserver.ErrPostingFailed
	}

	return nil
}
