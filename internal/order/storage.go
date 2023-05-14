package order

import "github.com/mserebryaakov/aggregator-order-service/pkg/postgres"

type Storage interface {
}

type OrderStorage struct {
	scp *postgres.SchemaConnectionPool
}

func NewStorage(scp *postgres.SchemaConnectionPool) Storage {
	return &OrderStorage{
		scp: scp,
	}
}
