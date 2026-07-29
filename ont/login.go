package ont

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type LoginResponse struct {
	SessionToken     string `json:"sess_token"`
	LoginNeedRefresh bool   `json:"login_need_refresh"`
}

func Login(endpoint, username, password string) (*Session, error) {
	jar, _ := cookiejar.New(nil)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	session := &Session{
		Client: &http.Client{
			Jar:       jar,
			Timeout:   15 * time.Second,
			Transport: transport,
		},
		Endpoint: endpoint,
	}

	// Get session token
	sessionToken, err := session.GetSessionToken()
	if err != nil {
		return nil, err
	}

	// Get login token
	loginToken, err := session.GetLoginToken()
	if err != nil {
		return nil, err
	}

	preparedHash := sha256.New()
	preparedHash.Write([]byte(password + loginToken))

	var payload url.Values = map[string][]string{
		"action":        {"login"},
		"Username":      {username},
		"Password":      {hex.EncodeToString(preparedHash.Sum(nil))},
		"_sessionTOKEN": {sessionToken},
	}

	resp, err := session.Post(endpoint+"/?_type=loginData&_tag=login_entry", "application/x-www-form-urlencoded; charset=UTF-8", strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.LoginNeedRefresh {
		refresh, err := session.Get(endpoint + "/")
		if err != nil {
			return nil, err
		}
		_ = refresh.Body.Close()
		return session, nil
	}
	return nil, errors.New("failed to login")
}
