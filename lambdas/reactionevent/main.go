package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/cbtexan04/willowbucks/db"
	"github.com/cbtexan04/willowbucks/slack"
)

var m = map[string]int{
	"willowbuck":   1,
	"willowbuck5":  5,
	"willowbuck10": 10,
}

const (
	ModeCredit = iota
	ModeTransfer
)

const (
	mode         = ModeCredit
	buckReaction = "willowbuck"
)

var (
	notificationChannel = os.Getenv("NOTIFY_CHANNEL")
)

var (
	ErrUnknownReaction = errors.New("unknown reaction")
	ErrUnknownSender   = errors.New("unknown sender")
	ErrUnknownReceiver = errors.New("unknown receiver")
	ErrUnknownChannel  = errors.New("unknown channel")
	ErrSelfPromotion   = errors.New("Self-tipping requests are ignored")
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
	var err error
	var from, to *slack.SlackUser

	amount, ok := m[event.Event.Reaction]
	if !ok {
		log.Println("unknown reaction")
		return ErrUnknownReaction
	}

	channel := event.Event.Item.Channel
	if channel == "" {
		log.Println("unknown channel")
		return ErrUnknownChannel
	}

	log.Printf("%+v", event)

	to, err = slack.UserLookup(event.Event.ItemUser)
	if err != nil {
		return err
	} else if to.User.ID == "" {
		return ErrUnknownReceiver
	}

	from, err = slack.UserLookup(event.Event.User)
	if err != nil {
		return err
	} else if from.User.ID == "" {
		return ErrUnknownSender
	}

	if to.User.ID == from.User.ID {
		slack.SendEphemeral(ErrSelfPromotion.Error(), to.User.ID, channel)
		log.Printf("%v: %+v", ErrSelfPromotion, event)
		return ErrSelfPromotion
	}

	switch mode {
	case ModeCredit:
		err := db.Credit(amount, to.User.ID)
		if err != nil {
			log.Printf("Unable to credit %d to [%s]: %v", amount, to.User.ID, err)
			return err
		}
	case ModeTransfer:
		err := db.Transfer(amount, from.User.ID, to.User.ID)
		if err != nil {
			if _, broke := err.(db.InsufficientFundErr); broke {
				notifyErr := slack.SendEphemeral(err.Error(), event.Event.User, event.Event.Item.Channel)
				if notifyErr != nil {
					log.Println("Unable to send notification:", notifyErr)
				}
			}

			log.Printf("Unable to credit %d from [%s] to [%s]: %v", amount, from.User.ID, to.User.ID, err)
			return err
		}

	default:
		return errors.New("Invalid mode")
	}

	var channelMsg string
	c, err := slack.ChannelLookup(channel)
	if err == nil {
		channelMsg = fmt.Sprintf("in channel %s", c.Channel.Name)
	}

	var msg string
	if amount == 1 {
		msg = fmt.Sprintf("%s sent a :willowbuck: to %s %s", from.User.RealName, to.User.RealName, channelMsg)
	} else {
		msg = fmt.Sprintf("%s sent %d :willowbuck: to %s %s", from.User.RealName, amount, to.User.RealName, channelMsg)
	}

	log.Println(msg)
	return slack.PostChannel(msg, notificationChannel)
}

func main() {
	lambda.Start(Handler)
}
