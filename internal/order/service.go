package order

import (
	"strconv"
	"strings"

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
	CreateSchema(schema string) error
	DeleteSchema(schema string) error
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
	var WaitingProcessingDelivery uint = 4
	var WaitingProcessingPayment uint = 1

	if order.ProductsIDs != "" {
		err := validateProductIDs(order.ProductsIDs)
		if err != nil {
			return 0, err
		}
	}

	newOrder := Order{
		UserID:           order.UserID,
		ProductsIDs:      order.ProductsIDs,
		DeliveryAddress:  order.DeliveryAddress,
		TotalPrice:       order.TotalPrice,
		AddressesShopID:  order.AddressesShopID,
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

func (s *orderService) CreateSchema(schema string) error {
	return s.storage.CreateSchema(schema)
}

func (s *orderService) DeleteSchema(schema string) error {
	return s.storage.DeleteSchema(schema)
}

func validateProductIDs(productIDs string) error {
	ids := strings.Split(productIDs, ",")
	for _, id := range ids {
		_, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return errProductIDsString
		}
	}
	return nil
}
