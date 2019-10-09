package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/cbtexan04/willowbucks/db"
	"github.com/cbtexan04/willowbucks/slack"
)

const topBalanceLimit = 5

type AccountEvent struct {
	Body string `json:"body"`
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

func myBalance(userID string) (*Response, error) {
	balance, err := db.GetBalance(userID, db.DefaultBalanceCredit)
	if err != nil {
		log.Println("unable to lookup balance:", err)
		return nil, err
	}

	return &Response{
		StatusCode: 200,
		Body:       fmt.Sprintf("You currently have %d :willowbuck:", balance),
	}, nil
}

func topBalances(event AccountEvent) (*Response, error) {
	accounts, err := db.GetTopBalances(topBalanceLimit)
	if err != nil {
		return nil, err
	}

	// TODO: user lookup on top balances

	resp := "Here's the top users by :willowbuck: balance:"
	for _, a := range accounts {
		u, err := slack.UserLookup(a.User)
		if err != nil {
			log.Println("Unable to lookup user %s: %v", a.User, err)
			continue
		}

		resp = fmt.Sprintf("%s\n%s: %d", resp, u.User.RealName, a.Balance)
	}

	fmt.Println(resp)

	return &Response{
		StatusCode: 200,
		Body:       resp,
	}, nil
}

func Handler(event AccountEvent) (*Response, error) {
	// Parse out all of the information from the body field into a map. This really
	// sucks but we're sent params which are url encoded in the body field of the
	// request, so we have to parse it out somehow
	m := make(map[string]string)
	tokens := strings.Split(event.Body, "&")
	for _, t := range tokens {
		if decoded, err := url.QueryUnescape(t); err == nil {
			t = decoded
		}

		split := strings.SplitN(t, "=", 2)
		if len(split) == 2 {
			m[split[0]] = split[1]
		}
	}

	log.Println(m["command"])

	// TODO: add a slash to the command
	if m["command"] == "/balance" {
		return myBalance(m["user_id"])
	} else {
		return topBalances(event)
	}
}

func main() {
	lambda.Start(Handler)
}
