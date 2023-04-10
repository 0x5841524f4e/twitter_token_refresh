package main

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"
)

func main() {
	fmt.Printf("Token refresher run.\n\n")
	var accessToken, refreshToken, clientID, clientSecret, refreshIntervnal string
	refreshToken = ""
	clientID = ""
	clientSecret = ""
	refreshIntervnal = "20"

	if refreshToken == "" {
		fmt.Println("err", "missing TWITTER_REFRESH_TOKEN variable")
		return
	}
	if clientID == "" || clientSecret == "" {
		fmt.Println("err", "missing variables")
		return
	}

	minutes, err := strconv.Atoi(refreshIntervnal)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	ticker := time.NewTicker(time.Duration(minutes) * time.Minute)
	var counter int
	for ; true; <-ticker.C {
		fmt.Printf("try to refresh token at %s using refresh_token %s...\n", time.Now().Format(time.RFC3339), refreshToken[:10])
		params := url.Values{}
		params.Set("refresh_token", refreshToken)
		params.Set("grant_type", "refresh_token")
		body := bytes.NewBufferString(params.Encode())

		client := &http.Client{}
		req, err := http.NewRequest("POST", "https://api.twitter.com/2/oauth2/token", body)
		if err != nil {
			fmt.Println("err", err)
			continue
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, clientSecret)))))

		reqDebug, err := httputil.DumpRequest(req, true)
		if err != nil {
			fmt.Println("err", err)
			continue
		}
		fmt.Println(string(reqDebug))

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("err", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("unexpected status code, expected %d got %d", http.StatusOK, resp.StatusCode)
			continue
		}

		var r refreshResponse
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			fmt.Println("err", err)
			continue
		}

		accessToken = r.AccessToken
		refreshToken = r.RefreshToken
		expiresIn = r.ExpiresIn
		fmt.Println("successfully refreshed token. new tokens stored for next call.")
		fmt.Println("access_token:", accessToken)
		fmt.Println("refresh_token:", refreshToken)
		fmt.Println("expires_in:", expiresIn)
		counter++

		if counter == 2 {
			triggered, err := triggerRateLimit(client, accessToken)
			if err != nil {
				fmt.Println("err", err)
				continue
			}
			if triggered {
				fmt.Println("triggered rate limit!")
				continue
			}
		}
	}
}

func triggerRateLimit(c *http.Client, accessToken string) (bool, error) {
	var reqCounter int
	for {
		if accessToken == "" {
			return false, errors.New("access token can't be empty")
		}
		req, err := http.NewRequest("GET", "https://api.twitter.com/2/tweets?ids=123219320715168,12783431091214", nil)
		if err != nil {
			return false, err
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		resp, err := c.Do(req)
		if err != nil {
			return false, err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return true, nil
		}
		if resp.StatusCode == http.StatusOK {
			fmt.Println("fetching tweets. retry count: ", reqCounter)
		} else {
			return false, fmt.Errorf("unexpected status code: %d, wanted 200", resp.StatusCode)
		}
		reqCounter++
		continue
	}
}

type refreshResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}
