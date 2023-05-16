package order

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type paymentAdapter struct {
	client      http.Client
	log         *logrus.Entry
	paymentHost string
	paymentPort string
}

func NewPaymentAdapter(log *logrus.Entry, paymentHost, paymentPort string) *paymentAdapter {
	c := http.Client{
		Timeout: time.Second * 10,
	}

	return &paymentAdapter{
		client:      c,
		log:         log,
		paymentHost: paymentHost,
		paymentPort: paymentPort,
	}
}

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Confirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

type Metadata struct {
	OrderID string `json:"order_id"`
}

type CreatePayment struct {
	Amount       Amount       `json:"amount"`
	Capture      bool         `json:"capture"`
	Confirmation Confirmation `json:"confirmation"`
	Description  string       `json:"description"`
	Metadata     Metadata     `json:"metadata"`
}

type PaymentConfirmation struct {
	Type            string `json:"type"`
	ConfirmationURL string `json:"confirmation_url"`
}

type Recipient struct {
	AccountID string `json:"account_id"`
	GatewayID string `json:"gateway_id"`
}

type Payment struct {
	ID           string              `json:"id"`
	Status       string              `json:"status"`
	Paid         bool                `json:"paid"`
	Amount       Amount              `json:"amount"`
	Confirmation PaymentConfirmation `json:"confirmation"`
	CreatedAt    time.Time           `json:"created_at"`
	Description  string              `json:"description"`
	Metadata     Metadata            `json:"metadata"`
	Recipient    Recipient           `json:"recipient"`
	Refundable   bool                `json:"refundable"`
	Test         bool                `json:"test"`
}

type GetPayment struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Paid        bool      `json:"paid"`
	Amount      Amount    `json:"amount"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	ExpiresAt   time.Time `json:"expires_at"`
	Metadata    Metadata  `json:"metadata"`
	Recipient   Recipient `json:"recipient"`
	Refundable  bool      `json:"refundable"`
	Test        bool      `json:"test"`
}

func (p *paymentAdapter) CreatePayment(createPayment CreatePayment) (*Payment, int, error) {
	createPaymentBytes, err := json.Marshal(createPayment)
	if err != nil {
		p.log.Debugf("error marshal createPayment")
	}

	url := fmt.Sprintf("http://%s%s%s", p.paymentHost, p.paymentPort, "/payment")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(createPaymentBytes))
	if err != nil {
		p.log.Errorf("failed create CreatePayment request - /payment - %v", err)
		return nil, 0, fmt.Errorf("failed CreatePayment request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Errorf("failed CreatePayment request:", err)
		return nil, 0, fmt.Errorf("failed CreatePayment request")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payment Payment
		err = json.NewDecoder(resp.Body).Decode(&payment)
		if err != nil {
			p.log.Errorf("failed to decode response body:", err)
			return nil, 500, fmt.Errorf("failed to decode response body")
		}
		return &payment, 200, nil
	case http.StatusBadRequest:
		p.log.Errorf("failed CreatePayment response (StatusBadRequest) :", err)
		return nil, 400, fmt.Errorf("failed CreatePayment response (StatusBadRequest)")
	default:
		p.log.Errorf("failed CreatePayment response (unexpected error) :", resp.StatusCode)
		return nil, 500, fmt.Errorf("failed CreatePayment response (unexpected error)")
	}
}

func (p *paymentAdapter) GetPayment(paymentId string) (*GetPayment, int, error) {
	url := fmt.Sprintf("http://%s%s%s", p.paymentHost, p.paymentPort, "/payment")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		p.log.Errorf("failed create GetPayment request - /payment - %v", err)
		return nil, 0, fmt.Errorf("failed GetPayment request")
	}

	q := req.URL.Query()
	q.Add("paymentId", paymentId)
	req.URL.RawQuery = q.Encode()

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Errorf("failed GetPayment request:", err)
		return nil, 0, fmt.Errorf("failed GetPayment request")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var payment GetPayment
		err = json.NewDecoder(resp.Body).Decode(&payment)
		if err != nil {
			p.log.Errorf("failed to decode response body:", err)
			return nil, 500, fmt.Errorf("failed to decode response body")
		}
		return &payment, 200, nil
	case http.StatusBadRequest:
		return nil, 400, fmt.Errorf("incorrect request (GetPayment request) - StatusBadRequest")
	case http.StatusNotFound:
		return nil, 404, fmt.Errorf("(GetPayment request) - StatusNotFound")
	default:
		return nil, 500, fmt.Errorf("unexpected status code (GetPayment request) - %v", resp.StatusCode)
	}
}
