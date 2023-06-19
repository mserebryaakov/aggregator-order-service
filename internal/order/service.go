package order

import (
	"github.com/sirupsen/logrus"
)

type OrderService interface {
	CreateOrder(order *Order, schema string) (uint, error)
	TakeOrderСourier(courierID, orderID uint, schema string) error
	DeliveredOrderСourier(courierID, orderID uint, schema string) error
	GetOrdersByUserID(userID uint, schema string) ([]Order, error)
	GetOrderByID(userId, orderId uint, schema string) (*Order, error)
	GetOrdersByDeliveryID(deliveryUserID uint, schema string) ([]Order, error)
	GetUnaxeptedOrderByAddressShopId(addressShopId []uint, schema string) ([]Order, error)
	UpdateOrderPaymentID(orderID uint, paymentID string, schema string) error
	GetOrderByPaymentKey(paymentKey string, schema string) (*Order, error)
	PaymentSuccess(orderID uint, schema string) error

	CreateCart(cart *Cart, schema string) (uint, error)
	GetCartWithProductsByUserID(userID uint, schema string) (*Cart, error)
	AddProductToCart(userID uint, product *Products, schema string) error
	RemoveProductFromCart(userID uint, product *Products, schema string) error
	ClearCartProducts(userID uint, schema string) error
}

type orderService struct {
	storage Storage
	logger  *logrus.Entry
}

func NewService(storage Storage, log *logrus.Entry) OrderService {
	return &orderService{
		storage: storage,
		logger:  log,
	}
}

func (s *orderService) CreateOrder(order *Order, schema string) (uint, error) {
	newOrder := Order{
		UserID:           order.UserID,
		Products:         order.Products,
		DeliveryAddress:  order.DeliveryAddress,
		TotalPrice:       order.TotalPrice,
		AddressesID:      order.AddressesID,
		PaymentKey:       order.PaymentKey,
		DeliveryStatusID: &WaitingProcessingDelivery,
		PaymentStatusID:  &WaitingProcessingPayment,
	}

	return s.storage.CreateOrder(&newOrder, schema)
}

func (s *orderService) TakeOrderСourier(courierID, orderID uint, schema string) error {
	return s.storage.TakeOrderСourier(courierID, orderID, schema)
}

func (s *orderService) DeliveredOrderСourier(courierID, orderID uint, schema string) error {
	return s.storage.DeliveredOrderСourier(courierID, orderID, schema)
}

func (s *orderService) GetOrdersByUserID(userID uint, schema string) ([]Order, error) {
	return s.storage.GetOrdersByUserID(userID, schema)
}

func (s *orderService) GetOrderByID(userId, orderId uint, schema string) (*Order, error) {
	return s.storage.GetOrderByID(userId, orderId, schema)
}

func (s *orderService) GetOrdersByDeliveryID(deliveryUserID uint, schema string) ([]Order, error) {
	return s.storage.GetOrdersByDeliveryID(deliveryUserID, schema)
}

func (s *orderService) GetUnaxeptedOrderByAddressShopId(addressShopId []uint, schema string) ([]Order, error) {
	return s.storage.GetUnaxeptedOrderByAddressShopId(addressShopId, schema)
}

func (s *orderService) UpdateOrderPaymentID(orderID uint, paymentID string, schema string) error {
	return s.storage.UpdateOrderPaymentID(orderID, paymentID, schema)
}

func (s *orderService) GetOrderByPaymentKey(paymentKey string, schema string) (*Order, error) {
	return s.storage.GetOrderByPaymentKey(paymentKey, schema)
}

func (s *orderService) PaymentSuccess(orderID uint, schema string) error {
	return s.storage.PaymentSuccess(orderID, schema)
}

func (s *orderService) CreateCart(cart *Cart, schema string) (uint, error) {
	return s.storage.CreateCart(cart, schema)
}

func (s *orderService) GetCartWithProductsByUserID(userID uint, schema string) (*Cart, error) {
	return s.storage.GetCartWithProductsByUserID(userID, schema)
}

func (s *orderService) AddProductToCart(userID uint, product *Products, schema string) error {
	return s.storage.AddProductToCart(userID, product, schema)
}

func (s *orderService) RemoveProductFromCart(userID uint, product *Products, schema string) error {
	return s.storage.RemoveProductFromCart(userID, product, schema)
}

func (s *orderService) ClearCartProducts(userID uint, schema string) error {
	return s.storage.ClearCartProducts(userID, schema)
}
