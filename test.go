package main

import (
	"log"

	"github.com/baderkha/library/pkg/email"
)

func main() {

	mailer := email.NewMockSender()
	err := mailer.SendEmail(&email.Content{
		FromUserFriendlyName: "your friendly neighborhood spiderman",
		From:                 "ahmad@baderkhan.org",
		To:                   "ahmad@baderkhna.org",
		ToFriendlyName:       "ahmad baderkhan",
		CC:                   []string{},
		Subject:              "wow me please",
		Body:                 `something cool 123`,
	})

	if err != nil {
		log.Fatal(err)
	}

}
