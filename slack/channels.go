package slack

import (
	"encoding/json"
	"net/http"
	"time"
)

const ChannelLookupHook = "https://slack.com/api/channels.info"

type SlackChannel struct {
	Channel struct {
		Name string `json:"name"`
	} `json:"channel"`
}

func ChannelLookup(channel string) (*SlackChannel, error) {
	req, err := http.NewRequest(http.MethodGet, ChannelLookupHook, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("token", AccessToken)
	q.Add("channel", channel)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	c := SlackChannel{}
	err = json.NewDecoder(resp.Body).Decode(&c)
	return &c, err
}
