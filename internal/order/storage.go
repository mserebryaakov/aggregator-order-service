package order

import (
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

func (s *OrderStorage) CreateOrder(order *Order, schema string) (uint, error) {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return 0, err
	}

	result := db.Create(&order)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to create order - %s", result.Error)
	}
	return order.ID, nil
}

func (s *OrderStorage) DeliveredOrderСourier(courierID uint, orderID uint, schema string) error {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return err
	}

	result := db.Model(&Order{}).Where("id = ? AND courier_id = ?", orderID, courierID).Update("delivery_status_id", 3)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errOrderWithCourierNotFound
	}
	return nil
}

func (s *OrderStorage) PaymentSuccess(orderID uint, schema string) error {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return err
	}

	return db.Model(&Order{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"payment_status_id":  2,
		"delivery_status_id": 1,
	}).Error
}

func (s *OrderStorage) TakeOrderСourier(courierID uint, orderID uint, schema string) error {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return err
	}

	result := db.Model(&Order{}).Where("id = ?", orderID).Updates(
		map[string]interface{}{"courier_id": courierID, "delivery_status_id": 2})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errTakeOrderNotFound
	}

	return nil
}

func (s *OrderStorage) GetOrdersByUserID(userID uint, schema string) ([]Order, error) {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return []Order{}, err
	}

	var orders []Order
	err = db.Where("user_id = ?", userID).Find(&orders).Error
	if err != nil {
		// if err == gorm.ErrRecordNotFound {
		// 	return []Order{}, errOrderNotFound
		// }
		return []Order{}, err
	}
	return orders, nil
}

func (s *OrderStorage) GetOrdersByDeliveryID(deliveryUserID uint, schema string) ([]Order, error) {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return []Order{}, err
	}

	var orders []Order
	err = db.Where("courier_id = ? AND delivery_status_id = ?", deliveryUserID, 2).Find(&orders).Error
	if err != nil {
		// if err == gorm.ErrRecordNotFound {
		// 	return []Order{}, errOrderNotFound
		// }
		return []Order{}, err
	}
	return orders, nil
}

func (s *OrderStorage) GetOrderByID(userId, orderID uint, schema string) (*Order, error) {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return nil, err
	}

	var order Order
	err = db.Where("user_id = ?", userId).First(&order, orderID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errOrderWithUserIdAndOrderIdNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (s *OrderStorage) GetUnaxeptedOrderByAddressShopId(addressShopId []uint, schema string) ([]Order, error) {
	db, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return nil, err
	}

	var order []Order

	if err = db.Where("addresses_shop_id IN (?) AND delivery_status_id = ?", addressShopId, 1).Find(&order).Error; err != nil {
		return []Order{}, err
	}

	return order, nil
}

func (s *OrderStorage) CreateSchema(domain string) error {
	publicschema, err := s.scp.GetConnectionPool("public")
	if err != nil {
		return err
	}

	tx := publicschema.Begin()
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

	err = tx.Commit().Error
	if err != nil {
		return err
	}

	resschema, err := s.scp.GetConnectionPool(domain)
	if err != nil {
		return err
	}

	err = RunSchemaMigration(resschema)
	if err != nil {
		fmt.Printf("Fatal create schema - failed migrations - %s", domain)
		return err
	}

	return nil
}

func (s *OrderStorage) DeleteSchema(domain string) error {
	publicschema, err := s.scp.GetConnectionPool("public")
	if err != nil {
		return err
	}

	if err := publicschema.Exec("DROP SCHEMA IF EXISTS " + domain + " CASCADE").Error; err != nil {
		return err
	}

	return nil
}

func (s *OrderStorage) UpdateOrderPaymentID(orderID uint, paymentID string, schema string) error {
	publicschema, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return err
	}

	result := publicschema.Model(&Order{}).Where("id = ?", orderID).Update("payment_id", paymentID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errChangePaymentIdNotFound
	}

	return nil
}

func (s *OrderStorage) GetOrderByPaymentKey(paymentKey string, schema string) (*Order, error) {
	publicschema, err := s.scp.GetConnectionPool(schema)
	if err != nil {
		return nil, err
	}

	var order Order
	result := publicschema.Where("payment_key = ?", paymentKey).First(&order)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errOrderWithPaymentKeyNotfound
	}
	return &order, nil
}
