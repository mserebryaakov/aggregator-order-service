package order

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type AuthAdapterLogHook struct{}

func (h *AuthAdapterLogHook) Fire(entry *logrus.Entry) error {
	entry.Message = "AuthAdapter: " + entry.Message
	return nil
}

func (h *AuthAdapterLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

type authAdapter struct {
	SystemToken string
	client      http.Client
	log         *logrus.Entry
	authHost    string
	authPort    string
}

func NewAuthAdapter(log *logrus.Entry, authHost, authPort string) *authAdapter {
	c := http.Client{
		Timeout: time.Second * 10,
	}

	return &authAdapter{
		client:   c,
		log:      log,
		authHost: authHost,
		authPort: authPort,
	}
}

func (a *authAdapter) Login(email, password, domain string) error {
	requestBody := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		Email:    email,
		Password: password,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		a.log.Errorf("login: failed to marshal login request body - %v", err)
		return NewError(JsonAppError, "failed to marshal login request body", 400, err)
	}

	a.log.Debugf("login: body - %s", jsonBody)

	//url := fmt.Sprintf("http://%s.%s%s%s", domain, a.authHost, a.authPort, "/auth/login")
	url := fmt.Sprintf("http://%s%s%s", a.authHost, a.authPort, "/system/auth/login")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		a.log.Debugf("login: failed to create login request with err - %v", err)
		return NewError(ServerAppError, "failed to create login request", 500, err)
	}

	q := req.URL.Query()
	q.Add("domain", domain)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Debugf("login: failed login request with err - %v", err)
		return NewError(ServerAppError, "failed login request", 500, err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		a.log.Debugf("login: failed readAll body - %v", err)
		return NewError(ServerAppError, "failed read body", 500, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		cookies := resp.Cookies()

		var token string
		for _, cookie := range cookies {
			if cookie.Name == "Authorization" {
				token = cookie.Value
				break
			}
		}

		if token == "" {
			a.log.Debug("login: token not found")
			return NewError(ServerAppError, "token not found (cookies from authservice)", 500, nil)
		}

		a.SystemToken = token

		a.log.Debugf("login: success login - %s", token)

		return nil
	case http.StatusBadRequest:
		return NewError(HttpError, "authservice /auth/login StatusBadRequest", 400, fmt.Errorf("body - %s", string(bts)))
	case http.StatusUnauthorized:
		return NewError(HttpError, "authservice /auth/login Unauthorized", 401, nil)
	case http.StatusForbidden:
		return NewError(HttpError, "authservice /auth/login Forbidden", 403, nil)
	default:
		return NewError(HttpError, "authservice /auth/login unexpected", 500, fmt.Errorf("statuscode - %d, body - %s", resp.StatusCode, string(bts)))
	}
}

func (a *authAdapter) Auth(role []string, clientToken, domain string) (int, uint, error) {
	url := fmt.Sprintf("http://%s%s%s", a.authHost, a.authPort, "/system/auth/validate")

	requestBody := struct {
		Role []string `json:"role"`
	}{
		Role: role,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		a.log.Errorf("auth: failed to marshal login request body - %v", err)
		return 0, 0, NewError(ServerAppError, "failed to marshal login request body", 500, err)
	}

	a.log.Debugf("auth: body - %s", jsonBody)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		a.log.Errorf("auth: failed create authservice request /auth/validate - %v", err)
		return 0, 0, NewError(ServerAppError, "failed create auth request /auth/validate", 500, err)
	}

	q := req.URL.Query()
	q.Add("domain", domain)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", a.SystemToken)
	req.Header.Set("X-System-Token", clientToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return 0, 0, NewError(ServerAppError, "failed authservice request", 500, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, 0, nil
	}

	var responseBody struct {
		UserID uint `json:"userId"`
	}

	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	if err != nil {
		a.log.Errorf("auth: failed to decode response body: %v", err)
		return 0, 0, NewError(ServerAppError, "failed to decode response body", 500, err)
	}

	return resp.StatusCode, responseBody.UserID, nil
}

func (a *authAdapter) Init(domain, password, email string) error {
	url := fmt.Sprintf("http://%s%s%s", a.authHost, a.authPort, "/init/start")

	requestBody := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		Email:    email,
		Password: password,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		a.log.Errorf("init: failed to marshal login request body - %v", err)
		return NewError(ServerAppError, "failed to marshal login request body", 500, err)
	}

	a.log.Debugf("init: body - %s", jsonBody)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		a.log.Errorf("init: failed create authservice init request init/start - %v", err)
		return NewError(ServerAppError, "failed create authservice init request init/start", 500, err)
	}

	req.Header.Set("Authorization", a.SystemToken)
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	q.Add("domain", domain)
	req.URL.RawQuery = q.Encode()

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Errorf("init: failed request - %v", err)
		return NewError(ServerAppError, "failed authservice init/start request", 500, err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		a.log.Debugf("init: failed to read response body - %v", err)
		return NewError(ServerAppError, "failed read body", 500, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return NewError(HttpError, "authservice init/start StatusBadRequest", 400, fmt.Errorf("body - %s", string(bts)))
	case http.StatusUnauthorized:
		return NewError(HttpError, "authservice /init/start Unauthorized", 401, nil)
	case http.StatusForbidden:
		return NewError(HttpError, "authservice /init/start Forbidden", 403, nil)
	default:
		return NewError(HttpError, "authservice /init/start unexpected", 500, fmt.Errorf("statuscode - %d, body - %s", resp.StatusCode, string(bts)))
	}
}

func (a *authAdapter) Rollback(domain string) error {
	url := fmt.Sprintf("http://%s%s%s", a.authHost, a.authPort, "/init/rollback")

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		a.log.Errorf("rollback: failed create authservice rollback request init/rollback - %v", err)
		return NewError(ServerAppError, "failed create authservice rollback request init/rollback", 500, err)
	}

	req.Header.Set("Authorization", a.SystemToken)

	q := req.URL.Query()
	q.Add("domain", domain)
	req.URL.RawQuery = q.Encode()

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Errorf("rollback: failed send authservice rollback request init/rollback - %v", err)
		return NewError(ServerAppError, "failed send authservice rollback request init/rollback", 500, err)
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		a.log.Debugf("rollback: failed to read response body - %v", err)
		return NewError(ServerAppError, "failed read body", 500, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return NewError(HttpError, "authservice /init/rollback StatusBadRequest", 400, fmt.Errorf("body - %s", string(bts)))
	case http.StatusUnauthorized:
		return NewError(HttpError, "authservice /init/rollback Unauthorized", 401, nil)
	case http.StatusForbidden:
		return NewError(HttpError, "authservice /init/rollback Forbidden", 403, nil)
	case http.StatusNotFound:
		return NewError(HttpError, "authservice /init/rollback domain not found", 404, nil)
	default:
		return NewError(HttpError, "authservice /init/rollback unexpected", 500, fmt.Errorf("statuscode - %d, body - %s", resp.StatusCode, string(bts)))
	}
}
