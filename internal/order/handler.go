package order

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	//systemRole   string = "system"
	clientRole   string = "client"
	deliveryRole string = "delivery"
	adminRole    string = "admin"
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
	order := router.Group("/order")
	{
		order.POST("", h.authWithRoleMiddleware([]string{adminRole, clientRole}), h.CreateOrder)
		order.POST("/take", h.authWithRoleMiddleware([]string{deliveryRole}), h.TakeOrderСourier)
		order.POST("/delivered", h.authWithRoleMiddleware([]string{deliveryRole}), h.DeliveredOrderСourier)
		order.GET("/all", h.authWithRoleMiddleware([]string{adminRole, clientRole}), h.GetOrdersByUserID)
		order.GET("", h.authWithRoleMiddleware([]string{adminRole, clientRole}), h.GetOrderByID)
		order.GET("/delivery", h.authWithRoleMiddleware([]string{deliveryRole}), h.GetUnaxeptedOrderByAddressShopId)
		order.GET("/delivery/my", h.authWithRoleMiddleware([]string{deliveryRole}), h.GetOrdersByDeliveryID)
	}
}

func (h *orderHandler) CreateOrder(c *gin.Context) {
	h.log.Debugf("handler start CreateOrder")

	domain, err := h.getDomain(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	var body Order = Order{}

	if c.Bind(&body) != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	_, err = h.orderService.CreateOrder(&body, domain)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *orderHandler) TakeOrderСourier(c *gin.Context) {
	h.log.Debugf("handler start TakeOrderСourier")

	orderIdStr := c.Query("orderId")
	if orderIdStr == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (orderId)")
		return
	}

	orderId, err := convertStringToUint(orderIdStr)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "query parameter orderId is not a id")
		return
	}

	domain, err := h.getDomain(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.orderService.TakeOrderСourier(userId, orderId, domain)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *orderHandler) DeliveredOrderСourier(c *gin.Context) {
	h.log.Debugf("handler start DeliveredOrderСourier")

	orderIdStr := c.Query("orderId")
	if orderIdStr == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (orderId)")
		return
	}

	orderId, err := convertStringToUint(orderIdStr)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "query parameter orderId is not a id")
		return
	}

	domain, err := h.getDomain(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.orderService.DeliveredOrderСourier(userId, orderId, domain)
	if err != nil {
		if err == errOrderWithCourierNotFound {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *orderHandler) GetOrdersByUserID(c *gin.Context) {
	h.log.Debugf("handler start GetOrdersByUserID")

	domain, err := h.getDomain(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	orders, err := h.orderService.GetOrdersByUserID(userId, domain)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *orderHandler) GetOrderByID(c *gin.Context) {
	h.log.Debugf("handler start GetOrderByID")

	orderIdStr := c.Query("orderId")
	if orderIdStr == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (orderId)")
		return
	}

	orderId, err := convertStringToUint(orderIdStr)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "query parameter orderId is not a id")
		return
	}

	domain, err := h.getDomain(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	order, err := h.orderService.GetOrderByID(userId, orderId, domain)
	if err != nil {
		if err == errOrderWithUserIdAndOrderIdNotFound {
			h.newErrorResponse(c, http.StatusNotFound, err.Error())
			return
		}
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *orderHandler) GetOrdersByDeliveryID(c *gin.Context) {
	h.log.Debugf("handler start GetOrdersByDeliveryID")

	domain, err := h.getDomain(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	orders, err := h.orderService.GetOrdersByDeliveryID(userId, domain)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *orderHandler) GetUnaxeptedOrderByAddressShopId(c *gin.Context) {
	h.log.Debugf("handler start GetUnaxeptedOrderByAddressShopId")

	var body struct {
		AddressShopId []uint `json:"addressShopId"`
	}

	if c.Bind(&body) != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	domain, err := h.getDomain(c)
	if err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	orders, err := h.orderService.GetUnaxeptedOrderByAddressShopId(body.AddressShopId, domain)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, orders)
}

type response struct {
	Message string `json:"message"`
}

func (h *orderHandler) newErrorResponse(c *gin.Context, statusCode int, message string) {
	h.log.Errorf(message)
	c.AbortWithStatusJSON(statusCode, &response{
		Message: message,
	})
}

func (h *orderHandler) getDomain(c *gin.Context) (string, error) {
	domain, exists := c.Get("domain")
	if exists {
		domainStr, ok := domain.(string)
		if ok {
			return domainStr, nil
		} else {
			return "", fmt.Errorf("incorrect domain type - %v", domain)
		}
	} else {
		return "", fmt.Errorf("domain not found")
	}
}

func (h *orderHandler) getUserId(c *gin.Context) (uint, error) {
	userId, exists := c.Get("userId")
	if exists {
		userIdStr, ok := userId.(string)
		if ok {
			userIdUint, err := convertStringToUint(userIdStr)
			if err != nil {
				return 0, fmt.Errorf("incorrect userId type - %v", userId)
			}
			return userIdUint, nil
		} else {
			return 0, fmt.Errorf("incorrect userId type - %v", userId)
		}
	} else {
		return 0, fmt.Errorf("userId not found")
	}
}

func (h *orderHandler) authWithRoleMiddleware(role []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host

		var shopDomain string
		if strings.Contains(host, ".") {
			arr := strings.Split(host, ".")
			if len(arr) == 2 {
				shopDomain = strings.Split(host, ".")[0]
			} else {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
		} else {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		tokenString := c.Request.Header.Get("Authorization")

		if tokenString == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		code, userId, err := h.authadapter.Auth(role, tokenString, shopDomain)
		if err != nil {
			h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}

		switch code {
		case 200:
			c.Set("domain", shopDomain)
			c.Set("userId", userId)
			c.Next()
		case 403:
			c.AbortWithStatus(http.StatusForbidden)
			return
		case 401:
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		case 404:
			c.AbortWithStatus(http.StatusNotFound)
			return
		default:
			h.log.Errorf("fatal unexpected auth result with code - %v", code)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}
}
