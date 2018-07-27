package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/dustin/go-nntp"
	"github.com/dustin/go-nntp/server"

	"github.com/mildred/newsweb/articles"
)

func convertGroup(grp *articles.Group) *nntp.Group {
	return &nntp.Group{
		Name:        grp.Name,
		Description: grp.Description,
		Count:       grp.Count,
		High:        grp.Low,
		Low:         grp.High,
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
		return nil, err // TODO: report internal error instead of closing the connection
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

func (s *Connection) GetArticle(group *nntp.Group, id string) (*nntp.Article, error) {
	num, errNum := strconv.Atoi(id)
	if errNum == nil {
		_ = num
		return nil, nntpserver.ErrInvalidArticleNumber
	} else {
		return nil, nntpserver.ErrInvalidMessageID
	}
}

func (s *Connection) GetArticles(group *nntp.Group, from, to int64) ([]nntpserver.NumberedArticle, error) {
	return nil, nil
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

func (s *Connection) Post(article *nntp.Article) error {
	groups := article.Header["Newsgroups"]
	if len(groups) == 0 {
		log.Print("ERROR: Newsgroup header absent")
		return nntpserver.ErrPostingFailed
	}

	// TODO: keep original articles with headers in same order
	var data bytes.Buffer
	for name, values := range article.Header {
		for _, val := range values {
			fmt.Fprintf(&data, "%s: %s\r\n", name, val)
		}
	}

	fmt.Fprint(&data, "\r\n")
	_, err := io.Copy(&data, article.Body)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nntpserver.ErrPostingFailed
	}

	var ok = true
	for _, group := range groups {
		_, err = s.Server.Articles.Post(group, data.Bytes())
		if err != nil {
			log.Printf("ERROR: %v", err)
			ok = false
		}
	}

	if !ok {
		return nntpserver.ErrPostingFailed
	}

	return nil
}
