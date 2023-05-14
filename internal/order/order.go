package order

import (
	"strconv"

	"github.com/sirupsen/logrus"
)

type OrderLogHook struct{}

func (h *OrderLogHook) Fire(entry *logrus.Entry) error {
	entry.Message = "Order: " + entry.Message
	return nil
}

func (h *OrderLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func convertStringToUint(str string) (uint, error) {
	floatVal, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	uintVal := uint(floatVal)
	return uintVal, nil
}
