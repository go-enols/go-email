package email

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func getAccessTokenFromRefreshToken(refreshToken, clientID string) (map[string]interface{}, error) {
	url := "https://login.microsoftonline.com/common/oauth2/v2.0/token"

	data := map[string]string{
		"client_id":     clientID,
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}

	formData := ""
	for k, v := range data {
		if formData != "" {
			formData += "&"
		}
		formData += fmt.Sprintf("%s=%s", k, v)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(formData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Host", "login.microsoftonline.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return nil, err
	}

	if tokenResp.Error == "" {
		return map[string]interface{}{
			"code":          0,
			"access_token":  tokenResp.AccessToken,
			"refresh_token": tokenResp.RefreshToken,
		}, nil
	}

	if strings.Contains(tokenResp.ErrorDescription, "User account is found to be in service abuse mode") {
		return map[string]interface{}{
			"code":    1,
			"message": "account was blocked or wrong username,password,refresh_token,client_id",
		}, nil
	}

	return map[string]interface{}{
		"code":    1,
		"message": "get access token is wrong",
	}, nil
}

// XOAUTH2Authenticator implements SASL XOAUTH2 authentication
type XOAUTH2Authenticator struct {
	Username    string
	AccessToken string
}

func (a *XOAUTH2Authenticator) Start() (mech string, ir []byte, err error) {
	authString := fmt.Sprintf("user=%s\001auth=Bearer %s\001\001", a.Username, a.AccessToken)
	return "XOAUTH2", []byte(authString), nil
}

func (a *XOAUTH2Authenticator) Next(challenge []byte) (response []byte, err error) {
	return nil, fmt.Errorf("unexpected challenge during XOAUTH2")
}
