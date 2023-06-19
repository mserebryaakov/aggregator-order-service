package order

import (
	"errors"
	"strings"
)

var (
	errOrderWithCourierNotFound          = errors.New("order with courierId not found")
	errOrderWithUserIdAndOrderIdNotFound = errors.New("order with userId and orderId not found")
	errTakeOrderNotFound                 = errors.New("not found order for take")
	errChangePaymentIdNotFound           = errors.New("order not found")
	errOrderWithPaymentKeyNotfound       = errors.New("order with paymentkey not found")
)

const (
	JsonAppError = iota
	ServerAppError
	HttpError
)

type ConstCodeAppError int

type AppError struct {
	Code     ConstCodeAppError
	Msg      string
	Err      error
	HTTPCode int
}

func NewError(code ConstCodeAppError, msg string, httpCode int, err error) error {
	return &AppError{
		Code:     code,
		Msg:      msg,
		Err:      err,
		HTTPCode: httpCode,
	}
}

func (ae *AppError) Error() string {
	b := new(strings.Builder)
	b.WriteString(ae.Msg + " ")
	if ae.Err != nil {
		b.WriteByte('(')
		b.WriteString(ae.Err.Error())
		b.WriteByte(')')
	}
	return b.String()
}

func (ae *AppError) Unwrap() error {
	return ae.Err
}
