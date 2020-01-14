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

const (
	WelcomeMsg = "Congratulations! You have been given your first willowbuck! You can send and receive willowbucks by using the :willowbuck: reaction on another user's message!\n\nYou can use the following commands at any time:\n/balance - check your current :willowbuck: balance\n/balances-top - see the top :willowbuck: earners"
)

var m = map[string]int{
	"willowbuck":   1,
	"willowbuck5":  5,
	"willowbuck10": 10,
}

const (
	ActionAddedEvent   = "reaction_added"
	ActionRemovedEvent = "reaction_removed"
	buckReaction       = "willowbuck"
)

var notificationChannel = os.Getenv("NOTIFY_CHANNEL")

var (
	ErrUnknownReaction  = errors.New("unknown reaction")
	ErrUnknownSender    = errors.New("unknown sender")
	ErrUnknownReceiver  = errors.New("unknown receiver")
	ErrUnknownChannel   = errors.New("unknown channel")
	ErrSelfPromotion    = errors.New("Self-tipping requests are ignored")
	ErrUnknownEventType = errors.New("unknown event type")
)

type reactionHandler func(ReactionEvent) error

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
	if _, ok := m[event.Event.Reaction]; !ok {
		// err, not a reaction we care about
	} else if event.Event.User == "" {
		// err, from user
	} else if event.Event.ItemUser == "" {
		// err, to user
	} else if event.Event.ItemUser == event.Event.User {
		// err, same user action
	} else if event.Event.Item.Channel == "" {
		// err, channel
	}

	// TODO: get the users/channel info, then pass along to the handler

	switch event.Event.Type {
	case ActionAddedEvent:
		return handleReactionAdded(event)
	case ActionRemovedEvent:
		return handleReactionRemoved(event)
	default:
		return ErrUnknownEventType
	}
}

func handleReactionRemoved(event ReactionEvent) error {
	// Not a reaction we care about
	if _, ok := m[event.Event.Reaction]; !ok {
		return nil
	}

	// TODO: come back and made this dynamic
	amount := 1

	to, err := slack.UserLookup(event.Event.ItemUser)
	if err != nil {
		return err
	} else if to.User.ID == "" {
		return ErrUnknownReceiver
	}

	from, err := slack.UserLookup(event.Event.User)
	if err != nil {
		return err
	} else if from.User.ID == "" {
		return ErrUnknownSender
	}

	channel := event.Event.Item.Channel
	channelInfo, err := slack.ChannelLookup(channel)
	if err != nil || channelInfo.Channel.Name == "" {
		log.Println("unknown channel")
		return ErrUnknownChannel
	}

	err = db.Debit(amount, to.User.ID)
	if err != nil {
		log.Printf("Unable to debit %d from [%s]: %v", amount, to.User.ID, err)
		return err
	}

	msg := fmt.Sprintf("%s removed their :willowbuck: from %s in #%s", from.User.RealName, to.User.RealName, channelInfo.Channel.Name)

	log.Println(msg)
	return slack.PostChannel(msg, notificationChannel)
}

func handleReactionAdded(event ReactionEvent) error {
	var err error
	var from, to *slack.SlackUser

	log.Printf("%+v", event)

	amount, ok := m[event.Event.Reaction]
	if !ok {
		log.Println("unknown reaction")
		return ErrUnknownReaction
	}

	channel := event.Event.Item.Channel
	channelInfo, err := slack.ChannelLookup(channel)
	if err != nil || channelInfo.Channel.Name == "" {
		log.Println("unknown channel")
		return ErrUnknownChannel
	}

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

	var userCreated bool
	userCreated, err = db.Credit(amount, to.User.ID)
	if err != nil {
		log.Printf("Unable to credit %d to [%s]: %v", amount, to.User.ID, err)
		return err
	}

	if userCreated {
		notifyErr := slack.SendEphemeral(WelcomeMsg, event.Event.ItemUser, event.Event.Item.Channel)
		if notifyErr != nil {
			log.Println("Unable to send notification:", notifyErr)
		}
	}

	var channelMsg string
	if err == nil && channelInfo.Channel.Name != "" {
		channelMsg = fmt.Sprintf("in channel #%s", channelInfo.Channel.Name)
	}

	var msg string
	if amount == 1 {
		if userCreated {
			msg = fmt.Sprintf("%s sent %s their first :willowbuck: %s", from.User.RealName, to.User.RealName, channelMsg)
		} else {
			msg = fmt.Sprintf("%s sent a :willowbuck: to %s %s", from.User.RealName, to.User.RealName, channelMsg)
		}
	} else {
		if userCreated {
			msg = fmt.Sprintf("%s sent %s their first %d :willowbuck: %s", from.User.RealName, to.User.RealName, amount, channelMsg)
		} else {
			msg = fmt.Sprintf("%s sent %d :willowbuck: to %s %s", from.User.RealName, amount, to.User.RealName, channelMsg)
		}
	}

	log.Println(msg)
	return slack.PostChannel(msg, notificationChannel)
}

func main() {
	lambda.Start(Handler)
}
