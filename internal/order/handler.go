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
	payment := router.Group("/payment")
	{
		payment.POST("/capture", h.authWithRoleMiddleware([]string{adminRole, clientRole}), h.CapturePayment) // old
		payment.POST("/cancel", h.authWithRoleMiddleware([]string{adminRole, clientRole}), h.CancelPayment)   // old
		payment.GET("", h.authWithRoleMiddleware([]string{adminRole, clientRole}), h.GetPayment)
		payment.POST("/refund", h.authWithRoleMiddleware([]string{adminRole, clientRole}), h.CreateRefund)
		payment.GET("/refund", h.authWithRoleMiddleware([]string{adminRole, clientRole}), h.GetRefund)
	}
	init := router.Group("/order/init")
	{
		init.POST("/start", h.authWithRoleMiddlewareSystem([]string{systemRole}), h.initstart)
		init.POST("/rollback", h.authWithRoleMiddlewareSystem([]string{systemRole}), h.initrollback)
	}
}

func (h *orderHandler) CheckRedirect(c *gin.Context) {
	h.log.Debugf("handler CheckRedirect")

	id := c.Param("id")
	domain := c.Param("domain")

	order, err := h.orderService.GetOrderByPaymentKey(id, domain)
	if err != nil {
		h.log.Errorf("CheckRedirect: GetOrderByPaymentKey err - %v", err)
		c.JSON(200, gin.H{})
		return
	}

	if order == nil {
		h.log.Errorf("CheckRedirect: order not found with id - %s, domain - %s ", id, domain)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	payment, _, err := h.paymentAdapter.GetPayment(order.PaymentID)
	if err != nil {
		h.log.Errorf("CheckRedirect: paymentAdapter GetPayment err - %v", err)
		c.JSON(200, gin.H{})
		return
	}

	if payment == nil {
		h.log.Errorf("CheckRedirect: payment not found (paymentId - %s)", order.PaymentID)
		c.JSON(200, gin.H{})
		return
	}

	if payment.Status == "succeeded" {
		err := h.orderService.PaymentSuccess(order.ID, domain)
		if err != nil {
			h.log.Errorf("CheckRedirect: PaymentSuccess err - %v", err)
			c.JSON(200, gin.H{})
			return
		}
	}

	c.JSON(200, gin.H{})
}

func (h *orderHandler) CreateOrder(c *gin.Context) {
	h.log.Debugf("handler CreateOrder")

	domain := h.getDomain(c)
	if domain == "" {
		h.log.Debug("CreateOrder: domain is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "domain is not defined")
		return
	}

	var body Order = Order{}

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Debugf("CreateOrder: failed to read body - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	h.log.Debugf("CreateOrder: body - %+v", body)

	userId, err := h.getUserId(c)
	if err != nil {
		h.log.Debug("CreateOrder: userId is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "userId is not defined")
		return
	}

	body.PaymentKey = uuid.New().String()
	body.UserID = userId

	orderId, err := h.orderService.CreateOrder(&body, domain)
	if err != nil {
		h.log.Debugf("CreateOrder: CreateOrder err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	returnUrl := fmt.Sprintf("%s%s%s/%s", h.redirectPath, "/redirect/", body.PaymentKey, domain)
	var createPayment CreatePayment = CreatePayment{
		Amount: Amount{
			Value:    strconv.FormatFloat(body.TotalPrice, 'f', 2, 64),
			Currency: "RUB",
		},
		Capture: true,
		Confirmation: &Confirmation{
			Type:      "redirect",
			ReturnUrl: &returnUrl,
		},
		Description: &body.DeliveryAddress,
		Metadata: Metadata{
			OrderID: strconv.FormatUint(uint64(orderId), 10),
		},
	}

	payment, _, err := h.paymentAdapter.CreatePayment(createPayment, body.PaymentKey)
	if err != nil {
		h.log.Debugf("CreateOrder: paymentAdapter CreatePayment err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, "failed create payment")
		return
	}

	err = h.orderService.UpdateOrderPaymentID(orderId, payment.ID, domain)
	if err != nil {
		h.log.Debugf("CreateOrder: UpdateOrderPaymentID err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, "update paymentId in orderIdFailed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": payment.Confirmation.ConfirmationURL,
	})
}

func (h *orderHandler) CapturePayment(c *gin.Context) {
	h.log.Debugf("handler CapturePayment")

	var body struct {
		PaymentID      string `json:"paymentID"`
		IdempotenceKey string `json:"idempotenceKey"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Debugf("CapturePayment: failed to read body - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	if body.PaymentID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "CapturePayment: PaymentID is not defined (body)")
	}

	if body.IdempotenceKey == "" {
		body.IdempotenceKey = uuid.New().String()
	}

	h.log.Debugf("CapturePayment: body - %+v", body)

	payment, httpcode, err := h.paymentAdapter.CapturePayment(body.IdempotenceKey, body.PaymentID)
	if err != nil {
		h.log.Debugf("CapturePayment: paymentAdapter CapturePayment err (http - %d) - %v", httpcode, err)
		h.newErrorResponse(c, http.StatusInternalServerError, "failed capture payment")
		return
	}

	c.JSON(http.StatusOK, payment)
}

func (h *orderHandler) CancelPayment(c *gin.Context) {
	h.log.Debugf("handler CancelPayment")

	var body struct {
		PaymentID      string `json:"paymentID"`
		IdempotenceKey string `json:"idempotenceKey"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Debugf("CancelPayment: failed to read body - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	if body.PaymentID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "CancelPayment: PaymentID is not defined (body)")
	}

	if body.IdempotenceKey == "" {
		body.IdempotenceKey = uuid.New().String()
	}

	h.log.Debugf("CancelPayment: body - %+v", body)

	payment, httpcode, err := h.paymentAdapter.CancelPayment(body.IdempotenceKey, body.PaymentID)
	if err != nil {
		h.log.Debugf("CancelPayment: paymentAdapter CancelPayment err (http - %d) - %v", httpcode, err)
		h.newErrorResponse(c, http.StatusInternalServerError, "failed cancel payment")
		return
	}

	c.JSON(http.StatusOK, payment)
}

func (h *orderHandler) CreateRefund(c *gin.Context) {
	h.log.Debugf("handler CreateRefund")

	var body struct {
		CreateRefund
		IdempotenceKey string `json:"idempotenceKey"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Debugf("CreateRefund: failed to read body - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	h.log.Debugf("CreateRefund: body - %+v", body)

	if body.IdempotenceKey == "" {
		body.IdempotenceKey = uuid.New().String()
	}

	refund, httpcode, err := h.paymentAdapter.CreateRefund(body.CreateRefund, body.IdempotenceKey)
	if err != nil {
		h.log.Debugf("CreateRefund: paymentAdapter CreateRefund err (httpcode - %d) - %v", httpcode, err)
		h.newErrorResponse(c, http.StatusInternalServerError, "failed create refund")
		return
	}

	c.JSON(http.StatusOK, refund)
}

func (h *orderHandler) GetRefund(c *gin.Context) {
	h.log.Debugf("handler GetRefund")

	var body struct {
		RefundId string `json:"refundId"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Debugf("GetRefund: failed to read body - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	h.log.Debugf("GetRefund: body - %+v", body)

	refund, httpcode, err := h.paymentAdapter.GetRefund(body.RefundId)
	if err != nil {
		h.log.Debugf("GetRefund: paymentAdapter GetRefund err (httpcode - %d) - %v", httpcode, err)
		h.newErrorResponse(c, http.StatusInternalServerError, "failed get refund")
		return
	}

	c.JSON(http.StatusOK, refund)
}

func (h *orderHandler) GetPayment(c *gin.Context) {
	h.log.Debugf("handler GetPayment")

	var body struct {
		PaymentId string `json:"paymentId"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Debugf("GetPayment: failed to read body - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	h.log.Debugf("GetPayment: body - %+v", body)

	payment, httpcode, err := h.paymentAdapter.GetPayment(body.PaymentId)
	if err != nil {
		h.log.Debugf("GetPayment: paymentAdapter GetPayment err (httpcode - %d) - %v", httpcode, err)
		h.newErrorResponse(c, http.StatusInternalServerError, "failed get payment")
		return
	}

	c.JSON(http.StatusOK, payment)
}

func (h *orderHandler) TakeOrderСourier(c *gin.Context) {
	h.log.Debugf("handler TakeOrderСourier")

	orderIdStr := c.Query("orderId")
	if orderIdStr == "" {
		h.log.Debug("TakeOrderСourier: orderId is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (orderId)")
		return
	}

	orderId, err := convertStringToUint(orderIdStr)
	if err != nil {
		h.log.Debugf("TakeOrderСourier: convertStringToUint err (orderIdStr - %v)", orderIdStr)
		h.newErrorResponse(c, http.StatusBadRequest, "query parameter orderId is not a id")
		return
	}

	domain := h.getDomain(c)
	if domain == "" {
		h.log.Debug("TakeOrderСourier: domain is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "domain is not defined")
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.log.Debugf("TakeOrderСourier: getUserId err - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.orderService.TakeOrderСourier(userId, orderId, domain)
	if err != nil {
		if err == errTakeOrderNotFound {
			h.log.Debug("TakeOrderСourier: TakeOrderСourier notfound")
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		h.log.Debugf("TakeOrderСourier: TakeOrderСourier err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *orderHandler) DeliveredOrderСourier(c *gin.Context) {
	h.log.Debugf("handler DeliveredOrderСourier")

	orderIdStr := c.Query("orderId")
	if orderIdStr == "" {
		h.log.Debug("DeliveredOrderСourier: orderId is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (orderId)")
		return
	}

	orderId, err := convertStringToUint(orderIdStr)
	if err != nil {
		h.log.Debugf("TakeOrderСourier: convertStringToUint err (orderIdStr - %v)", orderIdStr)
		h.newErrorResponse(c, http.StatusBadRequest, "query parameter orderId is not a id")
		return
	}

	domain := h.getDomain(c)
	if domain == "" {
		h.log.Debug("DeliveredOrderСourier: domain is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "domain is not defined")
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.log.Debugf("DeliveredOrderСourier: getUserId err - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.orderService.DeliveredOrderСourier(userId, orderId, domain)
	if err != nil {
		if err == errOrderWithCourierNotFound {
			h.log.Debug("DeliveredOrderСourier: DeliveredOrderСourier notfound")
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		h.log.Debugf("DeliveredOrderСourier: DeliveredOrderСourier err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *orderHandler) GetOrdersByUserID(c *gin.Context) {
	h.log.Debugf("handler GetOrdersByUserID")

	domain := h.getDomain(c)
	if domain == "" {
		h.log.Debug("GetOrdersByUserID: domain is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "domain is not defined")
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.log.Debugf("GetOrdersByUserID: getUserId err - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	orders, err := h.orderService.GetOrdersByUserID(userId, domain)
	if err != nil {
		h.log.Debugf("GetOrdersByUserID: GetOrdersByUserID err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *orderHandler) GetOrderByID(c *gin.Context) {
	h.log.Debugf("handler GetOrderByID")

	orderIdStr := c.Query("orderId")
	if orderIdStr == "" {
		h.log.Debug("GetOrderByID: orderId is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (orderId)")
		return
	}

	orderId, err := convertStringToUint(orderIdStr)
	if err != nil {
		h.log.Debugf("GetOrderByID: convertStringToUint err (orderIdStr - %v)", orderIdStr)
		h.newErrorResponse(c, http.StatusBadRequest, "query parameter orderId is not a id")
		return
	}

	domain := h.getDomain(c)
	if domain == "" {
		h.log.Debug("GetOrderByID: domain is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "domain is not defined")
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.log.Debugf("GetOrderByID: getUserId err - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	order, err := h.orderService.GetOrderByID(userId, orderId, domain)
	if err != nil {
		if err == errOrderWithUserIdAndOrderIdNotFound {
			h.log.Debug("GetOrderByID: GetOrderByID notfound")
			h.newErrorResponse(c, http.StatusNotFound, err.Error())
			return
		}
		h.log.Debugf("GetOrderByID: GetOrderByID err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *orderHandler) GetOrdersByDeliveryID(c *gin.Context) {
	h.log.Debugf("handler GetOrdersByDeliveryID")

	domain := h.getDomain(c)
	if domain == "" {
		h.log.Debug("GetOrdersByDeliveryID: domain is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "domain is not defined")
		return
	}

	userId, err := h.getUserId(c)
	if err != nil {
		h.log.Debugf("GetOrdersByDeliveryID: getUserId err - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	orders, err := h.orderService.GetOrdersByDeliveryID(userId, domain)
	if err != nil {
		h.log.Debugf("GetOrdersByDeliveryID: GetOrdersByDeliveryID err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *orderHandler) GetUnaxeptedOrderByAddressShopId(c *gin.Context) {
	h.log.Debugf("handler GetUnaxeptedOrderByAddressShopId")

	var body struct {
		AddressShopId []uint `json:"addressShopId"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Debugf("GetUnaxeptedOrderByAddressShopId: failed to read body - %v", err)
		h.newErrorResponse(c, http.StatusBadRequest, "failed to read body")
		return
	}

	h.log.Debugf("GetUnaxeptedOrderByAddressShopId: body - %+v", body)

	domain := h.getDomain(c)
	if domain == "" {
		h.log.Debug("GetOrdersByDeliveryID: domain is not defined")
		h.newErrorResponse(c, http.StatusBadRequest, "domain is not defined")
		return
	}

	orders, err := h.orderService.GetUnaxeptedOrderByAddressShopId(body.AddressShopId, domain)
	if err != nil {
		h.log.Debugf("GetUnaxeptedOrderByAddressShopId: GetUnaxeptedOrderByAddressShopId err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *orderHandler) initstart(c *gin.Context) {
	h.log.Debugf("handler initstart")

	domain := c.Query("domain")
	if domain == "" {
		h.log.Errorf("initstart: missing query parameter (domain)")
		h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (domain)")
		return
	}

	err := h.orderService.CreateSchema(domain)
	if err != nil {
		h.log.Debugf("initstart: CreateSchema err - %v", err)
		h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (h *orderHandler) initrollback(c *gin.Context) {
	h.log.Debugf("handler initrollback")

	domain := c.Query("domain")
	if domain == "" {
		h.log.Debug("initrollback: domain is not defined")
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

// Формирование respone
func (h *orderHandler) newErrorResponse(c *gin.Context, statusCode int, message string) {
	h.log.Errorf(message)
	c.AbortWithStatusJSON(statusCode, &response{
		Message: message,
	})
}

// Получение domain их контекста "domain"
func (h *orderHandler) getDomain(c *gin.Context) string {
	domain, exists := c.Get("domain")
	if exists {
		domainStr, ok := domain.(string)
		if ok {
			return domainStr
		}
		h.log.Errorf("incorrect domain type - %s", domain)
	}
	return ""
}

// Получение id пользователя из контекста "userId"
func (h *orderHandler) getUserId(c *gin.Context) (uint, error) {
	userId, exists := c.Get("userId")
	if exists {
		userIdUint, ok := userId.(uint)
		if ok {
			return userIdUint, nil
		}
		return 0, fmt.Errorf("incorrect userId type - %v", userId)
	} else {
		return 0, fmt.Errorf("userId not found")
	}
}

// Проверка домена (host) + Авторизация и аутентификация (jwt)
func (h *orderHandler) authWithRoleMiddleware(role []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h.log.Debug("handle authWithRoleMiddleware")

		host := c.Request.Host

		var shopDomain string
		if strings.Contains(host, ".") {
			arr := strings.Split(host, ".")
			if len(arr) == 2 {
				shopDomain = strings.Split(host, ".")[0]
			} else {
				h.log.Debug("authWithRoleMiddleware: host split error - len > 2")
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
		} else {
			h.log.Debug("authWithRoleMiddleware: domain is not defined (not contains `.`)")
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		tokenString := c.Request.Header.Get("Authorization")

		if tokenString == "" {
			h.log.Debug("authWithRoleMiddleware: authorization token not found")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		code, userId, err := h.authadapter.Auth(role, tokenString, shopDomain)
		if err != nil {
			h.log.Debugf("authWithRoleMiddleware: auth in authservice error - %v", err)
			h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}

		switch code {
		case 200:
			c.Set("domain", shopDomain)
			c.Set("userId", userId)
			c.Next()
		case 403:
			h.log.Debugf("authWithRoleMiddleware: auth in authservice with code - %d", 403)
			c.AbortWithStatus(http.StatusForbidden)
			return
		case 401:
			h.log.Debugf("authWithRoleMiddleware: auth in authservice with code - %d", 401)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		case 404:
			h.log.Debugf("authWithRoleMiddleware: auth in authservice with code - %d", 404)
			c.AbortWithStatus(http.StatusNotFound)
			return
		default:
			h.log.Debugf("authWithRoleMiddleware: auth in authservice with unexpected code - %d, err - %v", code, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}
}

// Проверка домена (query) + Авторизация и аутентификация (jwt)
func (h *orderHandler) authWithRoleMiddlewareSystem(role []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h.log.Debug("handle authWithRoleMiddlewareSystem")

		domain := c.Query("domain")
		if domain == "" {
			h.log.Errorf("authWithRoleMiddlewareSystem: missing query parameter (domain)")
			h.newErrorResponse(c, http.StatusBadRequest, "missing query parameter (domain)")
			return
		}

		tokenString := c.Request.Header.Get("Authorization")

		if tokenString == "" {
			h.log.Debug("authWithRoleMiddlewareSystem: authorization token not found")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		code, userId, err := h.authadapter.Auth(role, tokenString, domain)
		if err != nil {
			h.log.Debugf("authWithRoleMiddlewareSystem: auth in authservice error - %v", err)
			h.newErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}

		switch code {
		case 200:
			c.Set("domain", domain)
			c.Set("userId", userId)
			c.Next()
		case 403:
			h.log.Debugf("authWithRoleMiddlewareSystem: auth in authservice with code - %d", 403)
			c.AbortWithStatus(http.StatusForbidden)
			return
		case 401:
			h.log.Debugf("authWithRoleMiddlewareSystem: auth in authservice with code - %d", 401)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		case 404:
			h.log.Debugf("authWithRoleMiddlewareSystem: auth in authservice with code - %d", 404)
			c.AbortWithStatus(http.StatusNotFound)
			return
		default:
			h.log.Debugf("authWithRoleMiddlewareSystem: auth in authservice with unexpected code - %d, err - %v", code, err)
			h.log.Errorf("fatal unexpected auth result with code - %v", code)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}
}
