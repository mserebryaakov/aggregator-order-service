package order

import "errors"

var (
	errOrderWithCourierNotFound          = errors.New("order with courierId not found")
	errOrderWithUserIdAndOrderIdNotFound = errors.New("order with userId and orderId not found")
	//errOrderNotFound                     = errors.New("order not found")
	errProductIDsString            = errors.New("incorrect ProductsIDs")
	errTakeOrderNotFound           = errors.New("not found order for take")
	errChangePaymentIdNotFound     = errors.New("order not found")
	errOrderWithPaymentKeyNotfound = errors.New("order with paymentkey not found")
)
