package order

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type orderHandler struct {
	log          *logrus.Entry
	orderService OrderService
	authadapter  *authAdpater
}

func NewHandler(orderService OrderService, log *logrus.Entry, authadapter *authAdpater) *orderHandler {
	return &orderHandler{
		log:          log,
		orderService: orderService,
		authadapter:  authadapter,
	}
}

func (h *orderHandler) Register(router *gin.Engine) {

}
