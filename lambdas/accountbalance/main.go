package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/cbtexan04/willowbucks/db"
)

type AccountEvent map[string]interface{}

func Handler(event AccountEvent) error {
	// TODO: don't hardcode this
	userID := ""

	balance, err := db.GetBalance(userID)
	if err != nil {
		log.Println("unable to lookup balance:", err)
		return err
	}

	log.Println("balance: ", balance)

	// TODO: message the user as a bot

	return nil
}

func main() {
	lambda.Start(Handler)
}
