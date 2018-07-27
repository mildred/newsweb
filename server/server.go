package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/dustin/go-nntp/server"

	"github.com/mildred/newsweb/articles"
)

type Server struct {
	Articles   *articles.Articles
	ListenAddr string
}

func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// TODO: pass context
	a, err := net.ResolveTCPAddr("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("cannot resolve address %s, %v", s.ListenAddr, err)
	}

	// TODO: pass context
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return fmt.Errorf("cannot listen to address %s, %v", s.ListenAddr, err)
	}
	defer l.Close()

	for ctx.Err() == nil {
		// TODO: pass context
		c, err := l.AcceptTCP()
		if err != nil {
			log.Printf("Cannot accept connection: %v", err)
			continue
		}

		// TODO: pass context
		var cnx = &Connection{s}
		srv := nntpserver.NewServer(cnx)
		wg.Add(1)
		go func() {
			defer wg.Done()
			srv.Process(c)
		}()
	}

	return ctx.Err()
}
