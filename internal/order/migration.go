package order

import "gorm.io/gorm"

func RunSchemaMigration(db *gorm.DB) error {
	migrator := db.Migrator()

	if !migrator.HasTable(&PaymentStatus{}) {
		db.AutoMigrate(&PaymentStatus{})
		db.Create(&PaymentStatus{Name: "Ожидание оплаты", Code: "WaitingPayment"})
		db.Create(&PaymentStatus{Name: "Оплачено", Code: "SuccessPayment"})
		db.Create(&PaymentStatus{Name: "Отменено", Code: "CanceledPayment"})
	}

	if !migrator.HasTable(&DeliveryStatus{}) {
		db.AutoMigrate(&DeliveryStatus{})
		db.Create(&DeliveryStatus{Name: "Ожидание обработки", Code: "WaitingProcessing"})
		db.Create(&DeliveryStatus{Name: "В процессе доставки", Code: "ProcessOfDelivery"})
		db.Create(&DeliveryStatus{Name: "Доставлено", Code: "Delivered"})
		db.Create(&DeliveryStatus{Name: "Ожидание оплаты", Code: "WaitingPayment"})
	}

	if !migrator.HasTable(&Order{}) {
		db.AutoMigrate(&Order{})
	}

	return nil
}
