package main

import (
	"context"
	"flag"
	"log"
	"os"
	"sync"

	"github.com/mildred/newsweb/articles"
	"github.com/mildred/newsweb/server"
)

type Config struct {
	DataPath string
}

func main() {
	var wg = new(sync.WaitGroup)
	var ctx = context.Background()
	var art articles.Articles
	var srv server.Server

	srv.Articles = &art
	flag.StringVar(&art.StorageDir, "data", os.Getenv("NEWSWEB_DATA"), "Data path (NEWSWEB_DATA)")
	flag.StringVar(&srv.ListenAddr, "listen-nntp", ":119", "Listen address for NNTP server")
	flag.Parse()

	err := srv.Start(ctx, wg)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
