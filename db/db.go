package db

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var ErrBroke = errors.New("insufficient funds")

const (
	DefaultBalanceCredit   = 0
	DefaultBalanceTransfer = 5
)

type InsufficientFundErr struct {
	wrapped     error
	amount      int
	userbalance int
}

func (e InsufficientFundErr) Error() string {
	return fmt.Sprintf("Unable to send %d willowbucks (you have a balance of %d)", e.amount, e.userbalance)
}

var db = dynamodb.New(session.New(), aws.NewConfig().WithRegion("us-east-1"))

const TableName = "WillowTreeBank"

type Account struct {
	User    string `json:"user"`
	Balance int    `json:"balance"`
	NewUser bool   `json:"-"`
}

func GetAccount(user string, newUserBalance int) (*Account, error) {
	a := &Account{User: user}

	// Prepare the input for the query.
	input := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"user": {
				S: aws.String(user),
			},
		},
	}

	result, err := db.GetItem(input)
	if err != nil {
		return a, err
	} else if result.Item == nil {
		a.Balance = newUserBalance
		a.NewUser = true
		return a, nil
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, a)
	return a, err
}

func GetBalance(user string, defaultBalance int) (int, error) {
	a, err := GetAccount(user, defaultBalance)
	if err != nil {
		return -1, err
	} else if a == nil {
		return -1, errors.New("unknown account")
	}

	return a.Balance, err
}

// Update our count for a user
func updateAccount(a *Account) error {
	// TODO: should account for race conditions here; consider a lock
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":b": {
				N: aws.String(strconv.Itoa(a.Balance)),
			},
		},
		TableName: aws.String(TableName),
		Key: map[string]*dynamodb.AttributeValue{
			"user": {
				S: aws.String(a.User),
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set balance = :b"),
	}

	_, err := db.UpdateItem(input)
	return err
}

func Transfer(amount int, from, to string) (createdUser bool, err error) {
	fromUser, err := GetAccount(from, DefaultBalanceTransfer)
	if err != nil {
		return false, err
	}

	if fromUser.Balance < amount {
		return false, InsufficientFundErr{amount: amount, userbalance: fromUser.Balance}
	}

	toUser, err := GetAccount(to, DefaultBalanceTransfer)
	if err != nil {
		return toUser.NewUser, err
	}

	fromUser.Balance = (fromUser.Balance - amount)
	toUser.Balance = (toUser.Balance + amount)

	err = updateAccount(fromUser)
	if err != nil {
		return toUser.NewUser, err
	}

	err = updateAccount(toUser)
	if err != nil {
		return toUser.NewUser, err
	}

	return toUser.NewUser, nil
}

func Credit(amount int, to string) (createdUser bool, err error) {
	toUser, err := GetAccount(to, DefaultBalanceCredit)
	if err != nil {
		return toUser.NewUser, err
	}
	toUser.Balance = (toUser.Balance + amount)

	return toUser.NewUser, updateAccount(toUser)
}
