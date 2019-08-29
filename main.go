package main

import (
	"errors"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

// TODO: need permissions to message a specific user
// https://api.slack.com/methods/chat.postEphemeral

var m = map[string]int{
	"willowbuck1":  1,
	"willowbuck5":  5,
	"willowbuck10": 10,
}

const buckReaction = "willowbuck"

var (
	ErrUnknownReaction = errors.New("unknown reaction")
	ErrUnknownSender   = errors.New("unknown sender")
	ErrUnknownReceiver = errors.New("unknown receiver")
)

type ReactionEvent struct {
	Event struct {
		EventTs string `json:"event_ts"`
		Item    struct {
			Channel string `json:"channel"`
			Ts      string `json:"ts"`
			Type    string `json:"type"`
		} `json:"item"`
		ItemUser string `json:"item_user"`
		Reaction string `json:"reaction"`
		Type     string `json:"type"`
		User     string `json:"user"`
	} `json:"event"`
}

func Handler(event ReactionEvent) error {
	var amount int
	var ok bool
	var from, to string

	if amount, ok = m[event.Event.Reaction]; !ok {
		log.Println("unknown reaction")
		return ErrUnknownReaction
	}

	// Get who sent the reaction
	if from = event.Event.User; from == "" {
		log.Println("unknown sender")
		return ErrUnknownSender
	}

	// Get who received the reaction
	if to = event.Event.ItemUser; to == "" {
		log.Println("unknown receiver")
		return ErrUnknownReceiver
	}

	err := handleExchange(amount, from, to)
	if err != nil {
		// TODO: if insufficient funds, message the user
		log.Printf("Unable to credit %d from [%s] to [%s]: %v", amount, from, to, err)
		return err
	}

	log.Printf("User [%s] sent [%d] to [%s]", from, amount, to)
	return nil
}

func main() {
	lambda.Start(Handler)
}
