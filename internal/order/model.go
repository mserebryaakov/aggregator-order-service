package order

import "gorm.io/gorm"

type PaymentStatus struct {
	gorm.Model
	Code string
	Name string
}

type DeliveryStatus struct {
	gorm.Model
	Code string
	Name string
}

type Order struct {
	gorm.Model
	UserID           uint
	ProductsIDs      []uint
	DeliveryAddress  string
	TotalPrice       float64
	AddressesShopID  uint
	DeliveryStatusID *uint
	DeliveryStatus   DeliveryStatus `gorm:"foreignKey:DeliveryStatusID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PaymentStatusID  *uint
	PaymentStatus    PaymentStatus `gorm:"foreignKey:PaymentStatusID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CourierID        *uint
}
