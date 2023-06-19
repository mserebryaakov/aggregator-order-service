package order

import (
	"errors"

	"github.com/mserebryaakov/aggregator-order-service/pkg/postgres"
	"gorm.io/gorm"
)

type Storage interface {
	CreateOrder(order *Order, schema string) (uint, error)
	TakeOrderСourier(courierID uint, orderID uint, schema string) error
	DeliveredOrderСourier(courierID uint, orderID uint, schema string) error
	GetOrdersByUserID(userID uint, schema string) ([]Order, error)
	GetOrderByID(userId, orderID uint, schema string) (*Order, error)
	GetOrdersByDeliveryID(deliveryUserID uint, schema string) ([]Order, error)
	GetUnaxeptedOrderByAddressShopId(addressShopId []uint, schema string) ([]Order, error)
	UpdateOrderPaymentID(orderID uint, paymentID string, schema string) error
	GetOrderByPaymentKey(paymentKey string, schema string) (*Order, error)

	CreateCart(cart *Cart, schema string) (uint, error)
	GetCartWithProductsByUserID(userID uint, schema string) (*Cart, error)
	AddProductToCart(userID uint, product *Products, schema string) error
	RemoveProductFromCart(userID uint, product *Products, schema string) error
	ClearCartProducts(userID uint, schema string) error

	PaymentSuccess(orderID uint, schema string) error
}

type OrderStorage struct {
	scp *postgres.SchemaConnectionPool
}

func NewStorage(scp *postgres.SchemaConnectionPool) Storage {
	return &OrderStorage{
		scp: scp,
	}
}

func (s *OrderStorage) withConnectionPool(fn func(db *gorm.DB) error, schema string) error {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return err
	}
	return fn(db)
}

func (s *OrderStorage) CreateOrder(order *Order, schema string) (uint, error) {
	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Create(&order).Error
	}, schema)

	if err != nil {
		return 0, err
	}

	return order.ID, nil
}

func (s *OrderStorage) DeliveredOrderСourier(courierID uint, orderID uint, schema string) error {
	err := s.withConnectionPool(func(db *gorm.DB) error {
		result := db.Model(&Order{}).Where("id = ? AND courier_id = ?", orderID, courierID).Update("delivery_status_id", DeliveredDelivery)
		if result.Error == nil && result.RowsAffected == 0 {
			return errOrderWithCourierNotFound
		}
		return result.Error
	}, schema)

	return err
}

func (s *OrderStorage) PaymentSuccess(orderID uint, schema string) error {
	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Model(&Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
			"payment_status_id":  PaidPayment,
			"delivery_status_id": WaitingProcessing,
		}).Error
	}, schema)

	return err
}

func (s *OrderStorage) TakeOrderСourier(courierID uint, orderID uint, schema string) error {
	err := s.withConnectionPool(func(db *gorm.DB) error {
		result := db.Model(&Order{}).Where("id = ?", orderID).Updates(
			map[string]interface{}{"courier_id": courierID, "delivery_status_id": ProcessOfDelivery})

		if result.Error == nil && result.RowsAffected == 0 {
			return errTakeOrderNotFound
		}

		return result.Error
	}, schema)

	return err
}

func (s *OrderStorage) GetOrdersByUserID(userID uint, schema string) ([]Order, error) {
	var orders []Order

	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Where("user_id = ?", userID).Preload("Products").Find(&orders).Error
	}, schema)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *OrderStorage) GetOrdersByDeliveryID(deliveryUserID uint, schema string) ([]Order, error) {
	var orders []Order

	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Where("courier_id = ? AND delivery_status_id = ?", deliveryUserID, ProcessOfDelivery).Preload("Products").Find(&orders).Error
	}, schema)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *OrderStorage) GetOrderByID(userId, orderID uint, schema string) (*Order, error) {
	var order Order

	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Where("user_id = ?", userId).Preload("Products").First(&order, orderID).Error
	}, schema)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errOrderWithUserIdAndOrderIdNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (s *OrderStorage) GetUnaxeptedOrderByAddressShopId(addressShopId []uint, schema string) ([]Order, error) {
	var order []Order

	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Where("addresses_shop_id IN (?) AND delivery_status_id = ?", addressShopId, WaitingProcessing).Preload("Products").Find(&order).Error
	}, schema)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s *OrderStorage) UpdateOrderPaymentID(orderID uint, paymentID string, schema string) error {
	err := s.withConnectionPool(func(db *gorm.DB) error {
		result := db.Model(&Order{}).Where("id = ?", orderID).Update("payment_id", paymentID)
		if result.Error == nil && result.RowsAffected == 0 {
			return errChangePaymentIdNotFound
		}
		return result.Error
	}, schema)

	return err
}

func (s *OrderStorage) GetOrderByPaymentKey(paymentKey string, schema string) (*Order, error) {
	var order Order

	err := s.withConnectionPool(func(db *gorm.DB) error {
		result := db.Where("payment_key = ?", paymentKey).Preload("Products").First(&order)
		if result.Error == nil && result.RowsAffected == 0 {
			return errOrderWithPaymentKeyNotfound
		}
		return result.Error
	}, schema)

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (s *OrderStorage) CreateCart(cart *Cart, schema string) (uint, error) {
	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Create(&cart).Error
	}, schema)

	if err != nil {
		return 0, err
	}

	return cart.ID, nil
}

func (s *OrderStorage) GetCartWithProductsByUserID(userID uint, schema string) (*Cart, error) {
	var cart Cart

	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Where("user_id = ?", userID).Preload("Products").First(&cart).Error
	}, schema)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &cart, nil
}

func (s *OrderStorage) AddProductToCart(userID uint, product *Products, schema string) error {
	var cart Cart

	err := s.withConnectionPool(func(db *gorm.DB) error {
		getCartErr := db.Where("user_id = ?", userID).Preload("Products").First(&cart).Error
		if getCartErr != nil {
			return getCartErr
		}
		return db.Model(&cart).Association("Products").Append(product)
	}, schema)

	return err
}

func (s *OrderStorage) RemoveProductFromCart(userID uint, product *Products, schema string) error {
	var cart Cart

	err := s.withConnectionPool(func(db *gorm.DB) error {
		getCartErr := db.Where("user_id = ?", userID).Preload("Products").First(&cart).Error
		if getCartErr != nil {
			return getCartErr
		}
		return db.Model(&cart).Association("Products").Delete(product)
	}, schema)

	return err
}

func (s *OrderStorage) ClearCartProducts(userID uint, schema string) error {
	var cart Cart

	err := s.withConnectionPool(func(db *gorm.DB) error {
		getCartErr := db.Where("user_id = ?", userID).Preload("Products").First(&cart).Error
		if getCartErr != nil {
			return getCartErr
		}
		return db.Model(&cart).Association("Products").Clear()
	}, schema)

	return err
}
