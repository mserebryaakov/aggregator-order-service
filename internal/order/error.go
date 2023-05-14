package order

import "errors"

var (
	errOrderWithCourierNotFound          = errors.New("order with courierId not found")
	errOrderWithUserIdAndOrderIdNotFound = errors.New("order with userId and orderId not found")
	//errOrderNotFound                     = errors.New("order not found")
)
