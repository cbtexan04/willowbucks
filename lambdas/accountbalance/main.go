package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/cbtexan04/willowbucks/db"
	"github.com/cbtexan04/willowbucks/slack"
)

const topBalanceLimit = 5

var ErrUserNotFound = errors.New("user not found")

type AccountEvent struct {
	Body string `json:"body"`
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// TODO: this is really not optimal. We have to do a lookup
// in our database for UUIDs we already know about, and THEN
// do a query to slack for each user to see what their
// name is. Two inefficiencies, double the effort/cost. Since
// user NAMES won't change, can we store that in the dynamo
// datastore along with the account balance?

// Given a user's name, attempt to find their userID
func getUserIdFromText(name string) (string, error) {
	name = strings.TrimPrefix(name, "@")
	accounts, err := db.GetTopBalances(-1)
	if err != nil {
		return "", err
	}

	for _, a := range accounts {
		u, err := slack.UserLookup(a.User)
		if err != nil {
			log.Println("Unable to lookup user %s: %v", a.User, err)
			continue
		}

		if u.User.Name == name {
			return u.User.ID, nil
		}
	}

	return "", ErrUserNotFound
}

func getBalanceForUser(userID string) (*Response, error) {
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

	// TODO: we can clean some of this up if we were able to abstract out
	// the response body (get balance for user should NOT return the response
	// but rather the amount. Then in our handler we can make the response body
	switch {
	case m["text"] != "": // get user balance by display name
		id, err := getUserIdFromText(m["text"])
		if err == ErrUserNotFound {
			// Not an issue, the user just doesn't exist, so balance is 0
			return &Response{
				StatusCode: 200,
				Body:       fmt.Sprintf("%s currently has 0 :willowbuck:", m["text"]),
			}, nil
		} else if err != nil {
			return nil, err
		}

		balance, err := db.GetBalance(id, db.DefaultBalanceCredit)
		if err != nil {
			log.Println("unable to lookup balance:", err)
			return nil, err
		}

		return &Response{
			StatusCode: 200,
			Body:       fmt.Sprintf("%s currently has %d :willowbuck:", m["text"], balance),
		}, nil
	case m["command"] == "/willowbuck-balance": // get calling user's balance
		return getBalanceForUser(m["user_id"])
	default:
		return topBalances(event) // return top balances
	}
}

func main() {
	lambda.Start(Handler)
}
