package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/dustin/go-nntp/server"

	"github.com/mildred/newsweb/articles"
	"github.com/mildred/newsweb/mailer"
	"github.com/mildred/newsweb/validations"
)

type Server struct {
	Articles    *articles.Articles
	Validations *validations.Validations
	Mailer      *mailer.Mailer
	ListenAddr  string
}

func (s *Server) Start(ctx context.Context) error {
	var wg = new(sync.WaitGroup)
	err := s.Mailer.Start(ctx, wg)
	if err != nil {
		return err
	}

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

	go func() {
		<-ctx.Done()
		log.Print("INFO: Closing NNTP server...")
		l.Close()
	}()

	defer func() {
		log.Print("INFO: Waiting for client connections to close...")
		wg.Wait()
		log.Print("INFO: Client connections closed.")
	}()

	log.Printf("INFO: Started NNTP server on %s", s.ListenAddr)

	for ctx.Err() == nil {
		// TODO: pass context
		c, err := l.AcceptTCP()
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("ERROR: Cannot accept connection: %v", err)
			}
			continue
		}

		// TODO: pass context
		var cnx = &Connection{s}
		srv := nntpserver.NewServer(cnx)
		wg.Add(1)
		go func() {
			<-ctx.Done()
			c.Close()
		}()
		go func() {
			defer wg.Done()
			srv.Process(c)
		}()
	}

	return ctx.Err()
}
