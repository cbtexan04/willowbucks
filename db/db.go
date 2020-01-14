package db

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
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

func Debit(amount int, user string) error {
	toUser, err := GetAccount(user, DefaultBalanceCredit)
	if err != nil {
		return err
	}

	// If this is a new user or if would put the user into the negative, do nothing
	if toUser.NewUser || (toUser.Balance-amount) < 0 {
		// TODO: do we want to funnel this up as a non nil error?
		return nil
	}

	toUser.Balance = (toUser.Balance - amount)

	return updateAccount(toUser)
}

func GetTopBalances(limit int) ([]Account, error) {
	topBalances := make([]Account, 0)

	proj := expression.NamesList(expression.Name("user"), expression.Name("balance"))

	expr, err := expression.NewBuilder().WithProjection(proj).Build()
	if err != nil {
		return topBalances, err
	}
	// Build the query input parameters
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(TableName),
	}

	// Make the DynamoDB Query API call
	result, err := db.Scan(params)
	if err != nil {
		return topBalances, err
	}

	for _, i := range result.Items {
		a := Account{}
		err = dynamodbattribute.UnmarshalMap(i, &a)

		if err != nil {
			return topBalances, err
		}

		topBalances = append(topBalances, a)
	}

	// sort the accounts by balance, then truncate to the num of accounts desired
	sort.Slice(topBalances, func(i, j int) bool {
		return topBalances[i].Balance > topBalances[j].Balance
	})

	if len(topBalances) > limit {
		topBalances = topBalances[0:limit]
	}

	return topBalances, nil
}
