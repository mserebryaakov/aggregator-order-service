package order

import (
	"errors"
	"fmt"

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
	CreateSchema(domain string) error
	DeleteSchema(domain string) error
	UpdateOrderPaymentID(orderID uint, paymentID string, schema string) error
	GetOrderByPaymentKey(paymentKey string, schema string) (*Order, error)

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
		return db.Where("user_id = ?", userID).Find(&orders).Error
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
		return db.Where("courier_id = ? AND delivery_status_id = ?", deliveryUserID, ProcessOfDelivery).Find(&orders).Error
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
		return db.Where("user_id = ?", userId).First(&order, orderID).Error
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
		return db.Where("addresses_shop_id IN (?) AND delivery_status_id = ?", addressShopId, WaitingProcessing).Find(&order).Error
	}, schema)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s *OrderStorage) CreateSchema(domain string) error {
	err := s.withConnectionPool(func(db *gorm.DB) error {
		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}

		var count int64
		tx.Raw("SELECT COUNT(*) FROM pg_namespace WHERE nspname = ?", domain).Scan(&count)
		if count != 0 {
			tx.Rollback()
			return fmt.Errorf("create schema failed (already exists): %s", domain)
		}

		if err := tx.Exec("CREATE SCHEMA IF NOT EXISTS " + domain).Error; err != nil {
			tx.Rollback()
			return err
		}

		err := tx.Commit().Error
		if err != nil {
			return err
		}

		resschema, err := s.scp.GetConnectionPool(domain)
		if err != nil {
			return err
		}

		err = RunSchemaMigration(resschema)
		if err != nil {
			return err
		}

		return nil
	}, "public")

	return err
}

func (s *OrderStorage) DeleteSchema(domain string) error {
	err := s.withConnectionPool(func(db *gorm.DB) error {
		return db.Exec("DROP SCHEMA IF EXISTS " + domain + " CASCADE").Error
	}, "public")

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return err
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
		result := db.Where("payment_key = ?", paymentKey).First(&order)
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
