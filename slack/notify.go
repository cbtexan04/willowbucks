package slack

import (
	"bytes"
	"net/http"
	"time"
)

const (
	EphemeralHook = "https://slack.com/api/chat.postEphemeral"
	ChannelHook   = "https://slack.com/api/chat.postMessage"
	AccessToken   = ""
)

func SendEphemeral(msg string, user string, channel string) error {
	req, err := http.NewRequest(http.MethodPost, EphemeralHook, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("token", AccessToken)
	q.Add("user", user)
	q.Add("pretty", "1")
	q.Add("text", msg)
	q.Add("channel", channel)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return err
	}

	return nil
}

func PostChannel(msg string, channel string) error {
	req, err := http.NewRequest(http.MethodPost, ChannelHook, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("token", AccessToken)
	q.Add("pretty", "1")
	q.Add("text", msg)
	q.Add("channel", channel)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "ok" {
		return err
	}

	return nil
}
