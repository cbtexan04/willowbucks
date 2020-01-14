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

// Welcome message we send a user when he/she has received their first willowbuck
const WelcomeMsg = `Congratulations! You have been given your first willowbuck!

You can send and receive willowbucks by using the :willowbuck: reaction on
another user's message!

You can use the following commands at any time:
/willowbuck-balance      - check your current :willowbuck: balance
/willowbuck-top-balances - see the top :willowbuck: earners`

// Use a map to correlate the reaction name to an amount. This would allow
// us to add additional reactions with varying willowbuck amounts.
var m = map[string]int{
	"willowbuck":   1,
	"willowbuck5":  5,
	"willowbuck10": 10,
}

// Defined actions, as sent to us by slack
const (
	ActionAddedEvent   = "reaction_added"
	ActionRemovedEvent = "reaction_removed"
)

// Validation errors
var (
	ErrUnknownReaction  = errors.New("unknown reaction")
	ErrUnknownSender    = errors.New("unknown sender")
	ErrUnknownReceiver  = errors.New("unknown receiver")
	ErrUnknownChannel   = errors.New("unknown channel")
	ErrSelfPromotion    = errors.New("Self-reaction requests are ignored")
	ErrUnknownEventType = errors.New("unknown event type")
)

// OS env which indicates which channel (by ID) to post notification messages to
var notificationChannel = os.Getenv("NOTIFY_CHANNEL")

// ReactionEvent is a structure representing the data that's sent to us
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
	amount, ok := m[event.Event.Reaction]
	if !ok {
		return nil // err, not a reaction we care about
	} else if event.Event.User == "" {
		return ErrUnknownSender // err, from user
	} else if event.Event.ItemUser == "" {
		return ErrUnknownReceiver // err, to user
	} else if event.Event.Item.Channel == "" {
		return ErrUnknownChannel // err, channel
	}

	channel, err := slack.ChannelLookup(event.Event.Item.Channel)
	if err != nil || channel.Channel.Name == "" {
		log.Println("unknown channel")
		return ErrUnknownChannel
	}

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

	// Don't recognize actions that the user performs on themselves
	if event.Event.ItemUser == event.Event.User {
		slack.SendEphemeral(ErrSelfPromotion.Error(), to.User.ID, channel.Channel.ID)
		log.Printf("%v: %+v", ErrSelfPromotion, event)
		return ErrSelfPromotion
	}

	// Figure out if we want to credit or debit, based on the reaction event
	switch event.Event.Type {
	case ActionAddedEvent:
		return handleCredit(to, from, channel, amount)
	case ActionRemovedEvent:
		return handleDebit(to, from, channel, amount)
	default:
		return ErrUnknownEventType
	}
}

func handleDebit(to *slack.SlackUser, from *slack.SlackUser, channel *slack.SlackChannel, amount int) error {
	err := db.Debit(amount, to.User.ID)
	if err != nil {
		log.Printf("Unable to debit %d from [%s]: %v", amount, to.User.ID, err)
		return err
	}

	msg := fmt.Sprintf("%s removed their :willowbuck: from %s in #%s", from.User.RealName, to.User.RealName, channel.Channel.Name)

	log.Println(msg)
	return slack.PostChannel(msg, notificationChannel)
}

func handleCredit(to *slack.SlackUser, from *slack.SlackUser, channel *slack.SlackChannel, amount int) error {
	userCreated, err := db.Credit(amount, to.User.ID)
	if err != nil {
		log.Printf("Unable to credit %d to [%s]: %v", amount, to.User.ID, err)
		return err
	}

	// Send welcome message if this is the user's first willowbuck
	if userCreated {
		notifyErr := slack.SendEphemeral(WelcomeMsg, to.User.ID, channel.Channel.ID)
		if notifyErr != nil {
			log.Println("Unable to send notification:", notifyErr)
		}
	}

	channelMsg := fmt.Sprintf("in channel #%s", channel.Channel.Name)

	var msg string
	switch {
	case userCreated && amount == 1:
		msg = fmt.Sprintf("%s sent %s their first :willowbuck: %s", from.User.RealName, to.User.RealName, channelMsg)
	case userCreated && amount > 1:
		msg = fmt.Sprintf("%s sent %s their first %d :willowbuck: %s", from.User.RealName, to.User.RealName, amount, channelMsg)
	case !userCreated && amount == 1:
		msg = fmt.Sprintf("%s sent a :willowbuck: to %s %s", from.User.RealName, to.User.RealName, channelMsg)
	case !userCreated && amount > 1:
		msg = fmt.Sprintf("%s sent %d :willowbuck: to %s %s", from.User.RealName, amount, to.User.RealName, channelMsg)
	default:
		// This should never(™) happen
		msg = fmt.Sprintf("%s sent %d :willowbuck: to %s %s", from.User.RealName, amount, to.User.RealName, channelMsg)
	}

	log.Println(msg)
	return slack.PostChannel(msg, notificationChannel)
}

func main() {
	lambda.Start(Handler)
}
