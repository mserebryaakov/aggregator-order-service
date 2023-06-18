package order

import "gorm.io/gorm"

var (
	// delivery
	WaitingProcessing         uint = 1 // Ожидание обработки
	ProcessOfDelivery         uint = 2 // В процессе доставки
	DeliveredDelivery         uint = 3 // Доставлено
	WaitingProcessingDelivery uint = 4 // Ожидание оплаты

	// payment
	WaitingProcessingPayment uint = 1 // Ожидание оплаты
	PaidPayment              uint = 2 // Оплачено
	CanceledPayment          uint = 3 // Отменено
)

type PaymentStatus struct {
	gorm.Model
	Code string `json:"code"`
	Name string `json:"name"`
}

type DeliveryStatus struct {
	gorm.Model
	Code string `json:"code"`
	Name string `json:"name"`
}

type Order struct {
	gorm.Model
	UserID           uint           `json:"user_id"`
	ProductsIDs      string         `json:"products_ids"`
	DeliveryAddress  string         `json:"delivery_address"`
	TotalPrice       float64        `json:"total_price"`
	AddressesShopID  uint           `json:"addresses_shop_id"`
	PaymentID        string         `json:"payment_id"`
	PaymentKey       string         `json:"payment_key"`
	DeliveryStatusID *uint          `json:"delivery_status_id"`
	DeliveryStatus   DeliveryStatus `gorm:"foreignKey:DeliveryStatusID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PaymentStatusID  *uint          `json:"payment_status_id"`
	PaymentStatus    PaymentStatus  `gorm:"foreignKey:PaymentStatusID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	CourierID        *uint          `json:"courier_id"`
}
