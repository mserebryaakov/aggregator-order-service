package order

import "github.com/sirupsen/logrus"

type OrderLogHook struct{}

func (h *OrderLogHook) Fire(entry *logrus.Entry) error {
	entry.Message = "Order: " + entry.Message
	return nil
}

func (h *OrderLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
