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

type authAdapter struct {
	systemToken string
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
		a.log.Errorf("failed to marshal login request body:", err)
		return fmt.Errorf("failed to marshal login request body")
	}

	url := fmt.Sprintf("http://%s%s%s", a.authHost, a.authPort, "/system/auth/login")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		a.log.Errorf("failed to create login request:", err)
		return fmt.Errorf("failed to create login request")
	}

	q := req.URL.Query()
	q.Add("domain", domain)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Errorf("failed login request:", err)
		return fmt.Errorf("failed login request")
	}
	defer resp.Body.Close()

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
			a.log.Errorf("token not found")
			return fmt.Errorf("token not found")
		}

		a.systemToken = token

		fmt.Printf("LOGIN SUCCESS - %s", token)

		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("incorrect request (login request)")
	default:
		return fmt.Errorf("unexpected status code (login request) - %v", resp.StatusCode)
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
		a.log.Errorf("failed to marshal login request body:", err)
		return 0, 0, fmt.Errorf("failed to marshal login request body")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		a.log.Errorf("failed create auth request - /auth/validate")
		return 0, 0, fmt.Errorf("failed create auth request")
	}

	q := req.URL.Query()
	q.Add("domain", domain)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", a.systemToken)
	req.Header.Set("X-System-Token", clientToken)

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Errorf("failed auth request:", err)
		return 0, 0, fmt.Errorf("failed auth request")
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
		a.log.Errorf("failed to decode response body:", err)
		return 0, 0, fmt.Errorf("failed to decode response body")
	}

	return resp.StatusCode, responseBody.UserID, nil
}

func (a *authAdapter) Init(domain string) error {
	url := fmt.Sprintf("http://%s%s%s", a.authHost, a.authPort, "/init/start")
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		a.log.Errorf("failed create init request - init/start - %v", err)
		return fmt.Errorf("failed init auth request")
	}

	req.Header.Set("Authorization", a.systemToken)

	q := req.URL.Query()
	q.Add("domain", domain)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Errorf("failed init request:", err)
		return fmt.Errorf("failed init request")
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		a.log.Errorf("failed to read response body: %v", err)
		return fmt.Errorf("failed to read response body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("incorrect request (init request) - %+v", responseBody)
	default:
		return fmt.Errorf("unexpected status code (init request) - %v, %+v", resp.StatusCode, responseBody)
	}
}

func (a *authAdapter) Rollback(domain string) error {
	url := fmt.Sprintf("http://%s%s%s", a.authHost, a.authPort, "/init/rollback")
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		a.log.Errorf("failed create rollback request - init/rollback")
		return fmt.Errorf("failed rollback auth request")
	}

	req.Header.Set("Authorization", a.systemToken)
	req.Header.Set("Content-Type", "application/json")

	q := req.URL.Query()
	q.Add("domain", domain)
	req.URL.RawQuery = q.Encode()

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Errorf("failed rollback request:", err)
		return fmt.Errorf("failed rollback request")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("incorrect request (init request)")
	default:
		return fmt.Errorf("unexpected status code (login request) - %v", resp.StatusCode)
	}
}
