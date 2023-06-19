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

type Shop struct {
	gorm.Model
	Name        string `json:"name"`
	Description string `json:"description"`
	ContactInfo string `json:"contact_info"`
}

type Addresses struct {
	gorm.Model
	Region      string `json:"region"`
	City        string `json:"city"`
	Street      string `json:"street"`
	ContactInfo string `json:"contact_info"`
}

type Products struct {
	gorm.Model
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ImageID     string   `json:"image_id"`
	Price       uint     `json:"price"`
	CategoryID  *uint    `json:"category_id"`
	Category    Category `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type Category struct {
	gorm.Model
	ParentCategoryID *uint  `json:"parent_category_id"`
	Name             string `json:"name"`
	Path             string `json:"path"`
}

// Order service
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
	Products         []Products     `gorm:"many2many:order_products;" json:"products"`
	DeliveryAddress  string         `json:"delivery_address"`
	TotalPrice       float64        `json:"total_price"`
	AddressesID      int            `json:"addresses_id"`
	Addresses        Addresses      `gorm:"foreignKey:AddressesID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PaymentID        string         `json:"payment_id"`
	PaymentKey       string         `json:"payment_key"`
	DeliveryStatusID *uint          `json:"delivery_status_id"`
	DeliveryStatus   DeliveryStatus `gorm:"foreignKey:DeliveryStatusID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	PaymentStatusID  *uint          `json:"payment_status_id"`
	PaymentStatus    PaymentStatus  `gorm:"foreignKey:PaymentStatusID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	CourierID        *uint          `json:"courier_id"`
}

type Cart struct {
	gorm.Model
	UserID   uint       `json:"user_id"`
	Products []Products `gorm:"many2many:cart_products;" json:"products"`
}
