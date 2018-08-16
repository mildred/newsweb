package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/mildred/newsweb/articles"
	"github.com/mildred/newsweb/mailer"
	"github.com/mildred/newsweb/server"
	"github.com/mildred/newsweb/validations"
)

type Config struct {
	DataPath string
}

func main() {
	var ctx = mainContext()
	var art articles.Articles
	var val validations.Validations
	var srv server.Server
	var mail mailer.Mailer

	defaultPassFd, _ := strconv.Atoi(os.Getenv("NEWSWEB_SMTP_PASS_FD"))
	srv.Articles = &art
	srv.Validations = &val
	srv.Mailer = &mail
	flag.StringVar(&art.StorageDir, "data", os.Getenv("NEWSWEB_DATA"), "Data path (NEWSWEB_DATA)")
	flag.StringVar(&srv.ListenAddr, "listen-nntp", ":119", "Listen address for NNTP server")
	flag.StringVar(&mail.Mail, "email", "", "From e-mail")
	flag.StringVar(&mail.Host, "mail-server", "localhost", "SMTP/IMAP Hostname")
	flag.StringVar(&mail.SmtpPort, "smtp-port", "587", "SMTP submission port")
	flag.StringVar(&mail.ImapPort, "imap-port", "143", "IMAP server port")
	flag.StringVar(&mail.User, "mail-user", os.Getenv("NEWSWEB_MAIL_USER"), "SMTP/IMAP Username (NEWSWEB_MAIL_USER)")
	flag.StringVar(&mail.Pass, "mail-pass", os.Getenv("NEWSWEB_MAIL_PASS"), "SMTP/IMAP Password (NEWSWEB_MAIL_PASS)")
	flag.IntVar(&mail.PassFd, "mail-pass-fd", defaultPassFd, "SMTP Password from file-descriptor (NEWSWEB_MAIL_PASS_FD)")
	flag.StringVar(&mail.PassFile, "mail-pass-file", os.Getenv("NEWSWEB_SMTP_PASS_FILE"), "SMTP Password from file (NEWSWEB_MAIL_PASS_FILE)")
	flag.BoolVar(&mail.ImapDebug, "imap-debug", false, "IMAP debug")
	flag.Parse()
	val.StorageDir = art.StorageDir

	err := art.Open()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	defer art.Close()

	err = val.Open()
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	defer val.Close()

	err = srv.Start(ctx)
	if err != nil && ctx.Err() == nil {
		log.Fatalf("ERROR: %v", err)
	}
}
