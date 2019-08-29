package slack

import (
	"encoding/json"
	"net/http"
	"time"
)

const UserLookupHook = "https://slack.com/api/users.info"

type SlackUser struct {
	Ok   bool `json:"ok"`
	User struct {
		ID       string `json:"id"`
		TeamID   string `json:"team_id"`
		Name     string `json:"name"`
		Deleted  bool   `json:"deleted"`
		Color    string `json:"color"`
		RealName string `json:"real_name"`
		Tz       string `json:"tz"`
		TzLabel  string `json:"tz_label"`
		TzOffset int    `json:"tz_offset"`
		Profile  struct {
			Title                 string `json:"title"`
			Phone                 string `json:"phone"`
			Skype                 string `json:"skype"`
			RealName              string `json:"real_name"`
			RealNameNormalized    string `json:"real_name_normalized"`
			DisplayName           string `json:"display_name"`
			DisplayNameNormalized string `json:"display_name_normalized"`
			StatusText            string `json:"status_text"`
			StatusEmoji           string `json:"status_emoji"`
			StatusExpiration      int    `json:"status_expiration"`
			AvatarHash            string `json:"avatar_hash"`
			ImageOriginal         string `json:"image_original"`
			Email                 string `json:"email"`
			FirstName             string `json:"first_name"`
			LastName              string `json:"last_name"`
			Image24               string `json:"image_24"`
			Image32               string `json:"image_32"`
			Image48               string `json:"image_48"`
			Image72               string `json:"image_72"`
			Image192              string `json:"image_192"`
			Image512              string `json:"image_512"`
			Image1024             string `json:"image_1024"`
			StatusTextCanonical   string `json:"status_text_canonical"`
			Team                  string `json:"team"`
			IsCustomImage         bool   `json:"is_custom_image"`
		} `json:"profile"`
		IsAdmin           bool `json:"is_admin"`
		IsOwner           bool `json:"is_owner"`
		IsPrimaryOwner    bool `json:"is_primary_owner"`
		IsRestricted      bool `json:"is_restricted"`
		IsUltraRestricted bool `json:"is_ultra_restricted"`
		IsBot             bool `json:"is_bot"`
		IsAppUser         bool `json:"is_app_user"`
		Updated           int  `json:"updated"`
	} `json:"user"`
}

func UserLookup(user string) (*SlackUser, error) {
	req, err := http.NewRequest(http.MethodGet, UserLookupHook, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("token", AccessToken)
	q.Add("user", user)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	u := SlackUser{}
	err = json.NewDecoder(resp.Body).Decode(&u)
	return &u, err
}
