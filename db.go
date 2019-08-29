package main

import (
	"errors"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var db = dynamodb.New(session.New(), aws.NewConfig().WithRegion("us-east-1"))

var ErrInsufficientFunds = errors.New("insufficient funds")

const TableName = "WillowTreeBank"

type Account struct {
	User    string `json:"user"`
	Balance int    `json:"balance"`
}

func getAccount(user string) (*Account, error) {
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
		// If no matching item is found return 5.
		a.Balance = 5
		return a, nil
	}

	err = dynamodbattribute.UnmarshalMap(result.Item, a)
	return a, err
}

func getBalance(user string) (int, error) {
	a, err := getAccount(user)
	if err != nil {
		return -1, err
	} else if a == nil {
		return -1, errors.New("unknown account")
	}

	return a.Balance, err
}

// Update our count for a user
func updateAccount(a *Account) error {
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

func handleExchange(amount int, from, to string) error {
	fromUser, err := getAccount(from)
	if err != nil {
		return err
	}

	if fromUser.Balance < amount {
		return ErrInsufficientFunds
	}

	toUser, err := getAccount(to)
	if err != nil {
		return err
	}

	fromUser.Balance = (fromUser.Balance - amount)
	toUser.Balance = (toUser.Balance + amount)

	err = updateAccount(fromUser)
	if err != nil {
		return err
	}

	err = updateAccount(toUser)
	if err != nil {
		// TODO: issue here, because the from account got deducted, but
		// we couldnt update the new account. Maybe add an alert and
		// handle manually?
		return err
	}

	return nil
}
