package order

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	systemRole   string = "system"
	clientRole   string = "client"
	deliveryRole string = "delivery"
	adminRole    string = "admin"
)

type orderHandler struct {
	log            *logrus.Entry
	orderService   OrderService
	authadapter    *authAdapter
	paymentAdapter *paymentAdapter
	redirectPath   string
}

func NewHandler(orderService OrderService, log *logrus.Entry, authadapter *authAdapter, paymentAdapter *paymentAdapter, redirectPath string) *orderHandler {
	return &orderHandler{
		log:            log,
		orderService:   orderService,
		authadapter:    authadapter,
		paymentAdapter: paymentAdapter,
		redirectPath:   redirectPath,
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
		order.GET("/redirect/:id/:domain", h.CheckRedirect)
	}
	init := router.Group("/init")
	{
		init.POST("/start", h.authWithRoleMiddlewareSystem([]string{systemRole}), h.initstart)
		init.POST("/rollback", h.authWithRoleMiddlewareSystem([]string{systemRole}), h.initrollback)
	}
}

func (h *orderHandler) CheckRedirect(c *gin.Context) {
	id := c.Param("id")
	domain := c.Param("domain")

	order, err := h.orderService.GetOrderByPaymentKey(id, domain)
	if err != nil {
		h.log.Errorf("CheckRedirect error (GetOrderByPaymentKey) - %v", err)
		c.JSON(200, gin.H{})
	}

	payment, _, err := h.paymentAdapter.GetPayment(order.PaymentID)
	if err != nil {
		h.log.Errorf("CheckRedirect error (GetPayment) - %v", err)
		c.JSON(200, gin.H{})
	}

	if payment.Status == "succeeded" {
		err := h.orderService.PaymentSuccess(order.ID, domain)
		if err != nil {
			h.log.Errorf("CheckRedirect error - %v", err)
		}
	}

	c.JSON(200, gin.H{})
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

	body.PaymentKey = uuid.New().String()

	orderId, err := h.orderService.CreateOrder(&body, domain)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	var createPayment CreatePayment = CreatePayment{
		Amount: Amount{
			Value:    strconv.FormatFloat(body.TotalPrice, 'f', 2, 64),
			Currency: "RUB",
		},
		Capture: true,
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: fmt.Sprintf("%s%s%s/%s", h.redirectPath, "/redirect/", body.PaymentKey, domain),
		},
		Description: body.DeliveryAddress,
		Metadata: Metadata{
			OrderID: strconv.FormatUint(uint64(orderId), 10),
		},
	}

	payment, _, err := h.paymentAdapter.CreatePayment(createPayment)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "failed create payment")
		return
	}

	err = h.orderService.UpdateOrderPaymentID(orderId, payment.ID, domain)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "update paymentId in orderIdFailed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": payment.Confirmation.ConfirmationURL,
	})
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
		if err == errTakeOrderNotFound {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
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

func (h *orderHandler) initstart(c *gin.Context) {
	h.log.Debugf("login hadnler initstart")

	domain := c.Query("domain")
	if domain == "" {
		h.log.Errorf("initstart: missing query parameter (domain)")
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (domain)")
		return
	}

	err := h.orderService.CreateSchema(domain)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *orderHandler) initrollback(c *gin.Context) {
	h.log.Debugf("login hadnler initrollback")

	domain := c.Query("domain")
	if domain == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (domain)")
		return
	}

	err := h.orderService.DeleteSchema(domain)
	if err != nil {
		h.log.Errorf("initrollback: fatal error delete schema - %s", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})
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
		userIdUint, ok := userId.(uint)
		if ok {
			// userIdUint, err := convertStringToUint(userIdStr)
			// if err != nil {
			// 	return 0, fmt.Errorf("incorrect userId type - %v", userId)
			// }
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

func (h *orderHandler) authWithRoleMiddlewareSystem(role []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain := c.Query("domain")
		if domain == "" {
			h.log.Errorf("authWithRoleMiddlewareSystem: missing query parameter (domain)")
			h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (domain)")
			return
		}

		tokenString := c.Request.Header.Get("Authorization")

		if tokenString == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		code, userId, err := h.authadapter.Auth(role, tokenString, domain)
		if err != nil {
			h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}

		switch code {
		case 200:
			c.Set("domain", domain)
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
