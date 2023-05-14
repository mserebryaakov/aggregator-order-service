package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/mserebryaakov/aggregator-order-service/config"
	"github.com/mserebryaakov/aggregator-order-service/internal/order"
	"github.com/mserebryaakov/aggregator-order-service/pkg/httpserver"
	"github.com/mserebryaakov/aggregator-order-service/pkg/logger"
	"github.com/mserebryaakov/aggregator-order-service/pkg/postgres"
)

func main() {
	log := logger.NewLogger("debug", &logger.MainLogHook{})

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configs: %v", err)
	}

	env, err := config.GetEnvironment()
	if err != nil {
		log.Fatalf(err.Error())
	}

	orderLog := logger.NewLogger(env.LogLvl, &order.OrderLogHook{})
	postgresConfig := postgres.Config{
		Host:     env.PgHost,
		Port:     env.PgPort,
		Username: env.PgUser,
		Password: env.PgPassword,
		DBName:   env.PgDbName,
		SSLMode:  env.SSLMode,
		TimeZone: env.TimeZone,
	}

	scp := postgres.NewSchemaConnectionPool(postgresConfig, log)
	_, err = scp.GetConnectionPool("public")
	if err != nil {
		log.Fatalf("failed connection to db: %v", err)
	}

	orderRepository := order.NewStorage(scp)
	orderService := order.NewService(orderRepository, orderLog)
	authAdapter := order.NewAuthAdapter(orderLog, env.AuthHost, env.AuthPort)

	err = authAdapter.Login(env.SupervisorEmail, env.SupervisorHashPassword, "public")
	if err != nil {
		log.Fatalf("failed login in auth service: %v", err)
	}

	router := gin.New()

	orderHandler := order.NewHandler(orderService, orderLog, authAdapter)
	orderHandler.Register(router)

	server := new(httpserver.Server)

	go func() {
		if err := server.Run(cfg.Server.Port, router); err != nil {
			log.Fatal("Failed running server %v", err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	oscall := <-interrupt
	log.Infof("Shutdown server, %s", oscall)

	if err := server.Shutdown(context.Background()); err != nil {
		log.Errorf("Error occured on server shutting down: %v", err)
	}
}
