package order

import "github.com/sirupsen/logrus"

type OrderService interface {
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
