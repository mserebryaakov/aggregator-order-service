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

type PaymentAdapterLogHook struct{}

func (h *PaymentAdapterLogHook) Fire(entry *logrus.Entry) error {
	entry.Message = "PaymentAdapter: " + entry.Message
	return nil
}

func (h *PaymentAdapterLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

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

type Metadata struct {
	OrderID string `json:"order_id"`
}

type Recipient struct {
	AccountID string `json:"account_id"`
	GatewayID string `json:"gateway_id"`
}

type CancellationDatails struct {
	Party  string `json:"party"`
	Reason string `json:"reason"`
}

type Confirmation struct {
	Type            string  `json:"type"`
	ConfirmationURL *string `json:"confirmation_url"`
	ReturnUrl       *string `json:"return_url"`
	Locale          *string `json:"locale"`
}

type Payment struct {
	ID                  string               `json:"id"`
	Status              string               `json:"status"`
	Amount              Amount               `json:"amount"`
	Description         *string              `json:"description"`
	Recipient           Recipient            `json:"recipient"`
	CapturedAt          *time.Time           `json:"captured_at"`
	CreatedAt           time.Time            `json:"created_at"`
	ExpiresAt           *time.Time           `json:"expires_at"`
	Confirmation        *Confirmation        `json:"confirmation"`
	Test                bool                 `json:"test"`
	RefundedAmount      *Amount              `json:"refunded_amount"`
	Paid                bool                 `json:"paid"`
	Refundable          bool                 `json:"refundable"`
	Metadata            *Metadata            `json:"metadata"`
	CancellationDatails *CancellationDatails `json:"cancellation_details"`
	MerchantCustomerID  *string              `json:"merchant_customer_id"`
}

type CreatePayment struct {
	Amount             Amount        `json:"amount"`
	Description        *string       `json:"description"`
	Confirmation       *Confirmation `json:"confirmation"`
	Capture            bool          `json:"capture"`
	Metadata           Metadata      `json:"metadata"`
	MerchantCustomerID *string       `json:"merchant_customer_id"`
}

type Refund struct {
	ID                  string    `json:"id"`
	Status              string    `json:"status"`
	PaymentID           string    `json:"payment_id"`
	CancellationDetails *string   `json:"cancellation_details"`
	ReceiptRegistration *string   `json:"receipt_registration"`
	CreatedAt           time.Time `json:"created_at"`
	Amount              Amount    `json:"amount"`
	Description         *string   `json:"description"`
}

type CreateRefund struct {
	PaymentID string `json:"payment_id"`
	Amount    Amount `json:"amount"`
}

func (p *paymentAdapter) CreatePayment(createPayment CreatePayment, idempotenceKey string) (*Payment, int, error) {
	createPaymentBytes, err := json.Marshal(createPayment)
	if err != nil {
		p.log.Debugf("CreatePayment: error marshal createPayment - %v", err)
		return nil, 0, fmt.Errorf("error marshal createPayment")
	}

	url := fmt.Sprintf("http://%s%s%s", p.paymentHost, p.paymentPort, "/payment")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(createPaymentBytes))
	if err != nil {
		p.log.Errorf("CreatePayment: failed create CreatePayment request - /payment - %v", err)
		return nil, 0, fmt.Errorf("failed CreatePayment request")
	}

	q := req.URL.Query()
	q.Add("idempotenceKey", idempotenceKey)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Errorf("CreatePayment: failed CreatePayment request - %v", err)
		return nil, 0, fmt.Errorf("failed CreatePayment request")
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Debugf("CreatePayment: failed readAll body - %v", err)
		return nil, 0, fmt.Errorf("failed readAll body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var payment Payment
		buf := bytes.NewBuffer(bts)
		err = json.NewDecoder(buf).Decode(&payment)
		if err != nil {
			p.log.Errorf("CreatePayment: failed to decode response body - %v", err)
			return nil, 500, fmt.Errorf("failed to decode response body")
		}
		return &payment, 200, nil
	case http.StatusBadRequest:
		p.log.Errorf("CreatePayment: failed CreatePayment response (StatusBadRequest) - %v, body - %s", err, string(bts))
		return nil, 400, fmt.Errorf("failed CreatePayment response (StatusBadRequest)")
	default:
		p.log.Errorf("CreatePayment: failed CreatePayment response (unexpected error) - %d, body - %s", resp.StatusCode, string(bts))
		return nil, 500, fmt.Errorf("failed CreatePayment response (unexpected error)")
	}
}

func (p *paymentAdapter) GetPayment(paymentId string) (*Payment, int, error) {
	url := fmt.Sprintf("http://%s%s%s", p.paymentHost, p.paymentPort, "/payment")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		p.log.Errorf("GetPayment: failed create GetPayment request - /payment - %v", err)
		return nil, 0, fmt.Errorf("failed GetPayment request")
	}

	q := req.URL.Query()
	q.Add("paymentId", paymentId)
	req.URL.RawQuery = q.Encode()

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Errorf("GetPayment: failed GetPayment request - %v", err)
		return nil, 0, fmt.Errorf("failed GetPayment request")
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Debugf("GetPayment: failed readAll body - %v", err)
		return nil, 0, fmt.Errorf("failed readAll body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var payment Payment
		buf := bytes.NewBuffer(bts)
		err = json.NewDecoder(buf).Decode(&payment)
		if err != nil {
			p.log.Errorf("GetPayment: failed to decode response body - %v", err)
			return nil, 500, fmt.Errorf("failed to decode response body")
		}
		return &payment, 200, nil
	case http.StatusBadRequest:
		return nil, 400, fmt.Errorf("GetPayment: incorrect request (GetPayment request) - StatusBadRequest, body - %s", string(bts))
	case http.StatusNotFound:
		return nil, 404, fmt.Errorf("GetPayment: (GetPayment request) - StatusNotFound")
	default:
		return nil, 500, fmt.Errorf("GetPayment: unexpected status code (GetPayment request) - %v, body - %s", resp.StatusCode, string(bts))
	}
}

func (p *paymentAdapter) CapturePayment(idempotenceKey, paymentID string) (*Payment, int, error) {
	url := fmt.Sprintf("http://%s%s%s", p.paymentHost, p.paymentPort, "/payment/capture")
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		p.log.Errorf("CapturePayment: failed create CapturePayment request - /payment/capture - %v", err)
		return nil, 0, fmt.Errorf("failed CapturePayment request")
	}

	q := req.URL.Query()
	q.Add("idempotenceKey", idempotenceKey)
	q.Add("paymentID", paymentID)
	req.URL.RawQuery = q.Encode()

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Errorf("CapturePayment: failed CapturePayment request - %v", err)
		return nil, 0, fmt.Errorf("failed CapturePayment request")
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Debugf("CreatePayment: failed readAll body - %v", err)
		return nil, 0, fmt.Errorf("failed readAll body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var payment Payment
		buf := bytes.NewBuffer(bts)
		err = json.NewDecoder(buf).Decode(&payment)
		if err != nil {
			p.log.Errorf("CapturePayment: failed to decode response body - %v", err)
			return nil, 500, fmt.Errorf("failed to decode response body")
		}
		return &payment, 200, nil
	case http.StatusBadRequest:
		p.log.Errorf("CapturePayment: failed CapturePayment response (StatusBadRequest) - %v, body - %s", err, string(bts))
		return nil, 400, fmt.Errorf("failed CapturePayment response (StatusBadRequest)")
	default:
		p.log.Errorf("CapturePayment: failed CapturePayment response (unexpected error) - %d, body - %s", resp.StatusCode, string(bts))
		return nil, 500, fmt.Errorf("failed CapturePayment response (unexpected error)")
	}
}

func (p *paymentAdapter) CancelPayment(idempotenceKey, paymentID string) (*Payment, int, error) {
	url := fmt.Sprintf("http://%s%s%s", p.paymentHost, p.paymentPort, "/payment/cancel")
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		p.log.Errorf("CancelPayment: failed create CancelPayment request - /payment/cancel - %v", err)
		return nil, 0, fmt.Errorf("failed CancelPayment request")
	}

	q := req.URL.Query()
	q.Add("idempotenceKey", idempotenceKey)
	q.Add("paymentID", paymentID)
	req.URL.RawQuery = q.Encode()

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Errorf("CancelPayment: failed CancelPayment request - %v", err)
		return nil, 0, fmt.Errorf("failed CancelPayment request")
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Debugf("CancelPayment: failed readAll body - %v", err)
		return nil, 0, fmt.Errorf("failed readAll body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var payment Payment
		buf := bytes.NewBuffer(bts)
		err = json.NewDecoder(buf).Decode(&payment)
		if err != nil {
			p.log.Errorf("CancelPayment: failed to decode response body - %v", err)
			return nil, 500, fmt.Errorf("failed to decode response body")
		}
		return &payment, 200, nil
	case http.StatusBadRequest:
		p.log.Errorf("CancelPayment: failed CancelPayment response (StatusBadRequest) - %v, body - %s", err, string(bts))
		return nil, 400, fmt.Errorf("failed CancelPayment response (StatusBadRequest)")
	default:
		p.log.Errorf("CancelPayment: failed CancelPayment response (unexpected error) - %d, body - %s", resp.StatusCode, string(bts))
		return nil, 500, fmt.Errorf("failed CancelPayment response (unexpected error)")
	}
}

func (p *paymentAdapter) CreateRefund(createRefund CreateRefund, idempotenceKey string) (*Refund, int, error) {
	createRefundBytes, err := json.Marshal(createRefund)
	if err != nil {
		p.log.Debugf("CreateRefund: error marshal CreateRefund - %v", err)
		return nil, 0, fmt.Errorf("error marshal createRefund")
	}

	url := fmt.Sprintf("http://%s%s%s", p.paymentHost, p.paymentPort, "/refund")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(createRefundBytes))
	if err != nil {
		p.log.Errorf("CreateRefund: failed create CreateRefund request - /refund - %v", err)
		return nil, 0, fmt.Errorf("failed CreateRefund request")
	}

	q := req.URL.Query()
	q.Add("idempotenceKey", idempotenceKey)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Errorf("CreateRefund: failed CreateRefund request - %v", err)
		return nil, 0, fmt.Errorf("failed CreateRefund request")
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Debugf("CreateRefund: failed readAll body - %v", err)
		return nil, 0, fmt.Errorf("failed readAll body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var refund Refund
		buf := bytes.NewBuffer(bts)
		err = json.NewDecoder(buf).Decode(&refund)
		if err != nil {
			p.log.Errorf("CreateRefund: failed to decode response body - %v", err)
			return nil, 500, fmt.Errorf("failed to decode response body")
		}
		return &refund, 200, nil
	case http.StatusBadRequest:
		p.log.Errorf("CreateRefund: failed CreateRefund response (StatusBadRequest) - %v, body - %s", err, string(bts))
		return nil, 400, fmt.Errorf("failed CreateRefund response (StatusBadRequest)")
	default:
		p.log.Errorf("CreateRefund: failed CreateRefund response (unexpected error) - %d, body - %s", resp.StatusCode, string(bts))
		return nil, 500, fmt.Errorf("failed CreateRefund response (unexpected error)")
	}
}

func (p *paymentAdapter) GetRefund(refundId string) (*Refund, int, error) {
	url := fmt.Sprintf("http://%s%s%s", p.paymentHost, p.paymentPort, "/refund")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		p.log.Errorf("GetRefund: failed create GetRefund request - /refund - %v", err)
		return nil, 0, fmt.Errorf("failed GetRefund request")
	}

	q := req.URL.Query()
	q.Add("refundId", refundId)
	req.URL.RawQuery = q.Encode()

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Errorf("GetRefund: failed GetRefund request - %v", err)
		return nil, 0, fmt.Errorf("failed GetRefund request")
	}
	defer resp.Body.Close()

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Debugf("CreatePayment: failed readAll body - %v", err)
		return nil, 0, fmt.Errorf("failed readAll body")
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var refund Refund
		buf := bytes.NewBuffer(bts)
		err = json.NewDecoder(buf).Decode(&refund)
		if err != nil {
			p.log.Errorf("GetRefund: failed to decode response body - %v", err)
			return nil, 500, fmt.Errorf("failed to decode response body")
		}
		return &refund, 200, nil
	case http.StatusBadRequest:
		return nil, 400, fmt.Errorf("GetRefund: incorrect request (GetRefund request) - StatusBadRequest, body - %s", string(bts))
	case http.StatusNotFound:
		return nil, 404, fmt.Errorf("GetRefund: (GetRefund request) - StatusNotFound")
	default:
		return nil, 500, fmt.Errorf("GetRefund: unexpected status code (GetRefund request) - %v, body - %s", resp.StatusCode, string(bts))
	}
}
