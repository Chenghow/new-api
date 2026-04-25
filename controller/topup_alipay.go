package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	alipay "github.com/smartwalle/alipay/v3"
	"github.com/thanhpk/randstr"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const (
	PaymentMethodAlipay = "alipay"
)

func isAlipayV3Enabled() bool {
	return setting.AlipayEnabled &&
		strings.TrimSpace(setting.AlipayAppId) != "" &&
		strings.TrimSpace(setting.AlipayPrivateKey) != "" &&
		strings.TrimSpace(setting.AlipayPublicKey) != ""
}

func normalizeAlipayKey(key string) string {
	normalized := strings.TrimSpace(key)
	normalized = strings.ReplaceAll(normalized, "\\n", "\n")
	return normalized
}

func getAlipayClient() (*alipay.Client, error) {
	if !isAlipayV3Enabled() {
		return nil, errors.New("支付宝支付未启用或配置不完整")
	}
	client, err := alipay.New(
		strings.TrimSpace(setting.AlipayAppId),
		normalizeAlipayKey(setting.AlipayPrivateKey),
		!setting.AlipaySandbox,
	)
	if err != nil {
		return nil, err
	}
	if err = client.LoadAliPayPublicKey(normalizeAlipayKey(setting.AlipayPublicKey)); err != nil {
		return nil, err
	}
	return client, nil
}

func buildAlipayURL(base string, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	return base + path
}

func getAlipayNotifyURL() string {
	return buildAlipayURL(service.GetCallbackAddress(), "/api/alipay/notify")
}

func getAlipayTopupReturnURL() string {
	return buildAlipayURL(system_setting.ServerAddress, "/console/log")
}

func getAlipaySubscriptionReturnURL() string {
	return buildAlipayURL(system_setting.ServerAddress, "/console/topup")
}

func requestAlipayPagePayLink(subject, outTradeNo string, totalAmount float64, returnURL string) (string, error) {
	client, err := getAlipayClient()
	if err != nil {
		return "", err
	}

	payURL, err := client.TradePagePay(alipay.TradePagePay{
		Trade: alipay.Trade{
			NotifyURL:      getAlipayNotifyURL(),
			ReturnURL:      returnURL,
			Subject:        subject,
			OutTradeNo:     outTradeNo,
			TotalAmount:    strconv.FormatFloat(totalAmount, 'f', 2, 64),
			ProductCode:    "FAST_INSTANT_TRADE_PAY",
			TimeoutExpress: "15m",
		},
		IntegrationType: "PCWEB",
	})
	if err != nil {
		return "", err
	}
	if payURL == nil {
		return "", errors.New("支付宝支付链接为空")
	}
	return payURL.String(), nil
}

// RequestAlipayTopupPay handles Alipay v3 topup creation and returns true when request is handled.
func RequestAlipayTopupPay(c *gin.Context, req *EpayRequest, payMoney float64) bool {
	if req.PaymentMethod != PaymentMethodAlipay || !isAlipayV3Enabled() {
		return false
	}
	if payMoney < 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return true
	}

	id := c.GetInt("id")
	tradeNo := fmt.Sprintf("ALIUSR%dNO%s%d", id, common.GetRandomString(6), time.Now().Unix())

	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}

	topUp := &model.TopUp{
		UserId:        id,
		Amount:        amount,
		Money:         payMoney,
		TradeNo:       tradeNo,
		PaymentMethod: PaymentMethodAlipay,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return true
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 创建订单 trade_no=%s notify_url=%q return_url=%q", tradeNo, getAlipayNotifyURL(), getAlipayTopupReturnURL()))
	payLink, err := requestAlipayPagePayLink(
		fmt.Sprintf("TUC%d", req.Amount),
		tradeNo,
		payMoney,
		getAlipayTopupReturnURL(),
	)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 拉起支付失败 error=%q", err.Error()))
		topUp.Status = common.TopUpStatusFailed
		_ = topUp.Update()
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return true
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payLink,
			"order_id": tradeNo,
		},
	})
	return true
}

// RequestAlipaySubscriptionPay handles Alipay v3 subscription payment creation.
func RequestAlipaySubscriptionPay(c *gin.Context, plan *model.SubscriptionPlan) bool {
	if !isAlipayV3Enabled() || plan == nil {
		return false
	}

	userId := c.GetInt("id")
	tradeNo := fmt.Sprintf("SUBALIUSR%dNO%s%d", userId, randstr.String(4), time.Now().Unix())

	order := &model.SubscriptionOrder{
		UserId:        userId,
		PlanId:        plan.Id,
		Money:         plan.PriceAmount,
		TradeNo:       tradeNo,
		PaymentMethod: PaymentMethodAlipay,
		CreateTime:    time.Now().Unix(),
		Status:        common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return true
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("支付宝 创建订单 trade_no=%s notify_url=%q return_url=%q", tradeNo, getAlipayNotifyURL(), getAlipayTopupReturnURL()))
	payLink, err := requestAlipayPagePayLink(
		fmt.Sprintf("SUB:%s", plan.Title),
		tradeNo,
		plan.PriceAmount,
		getAlipaySubscriptionReturnURL(),
	)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 拉起订阅支付失败 error=%q", err.Error()))
		_ = model.ExpireSubscriptionOrder(tradeNo, PaymentMethodAlipay)
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起支付失败"})
		return true
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": payLink,
			"order_id": tradeNo,
		},
	})
	return true
}

func AlipayNotify(c *gin.Context) {
	ctx := c.Request.Context()
	logger.LogInfo(ctx, fmt.Sprintf("支付宝 webhook 收到请求 path=%q client_ip=%s method=%s", c.Request.RequestURI, c.ClientIP(), c.Request.Method))

	client, err := getAlipayClient()
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝 webhook 配置错误 client_ip=%s error=%q", c.ClientIP(), err.Error()))
		c.String(http.StatusOK, "fail")
		return
	}

	if err = c.Request.ParseForm(); err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝 webhook 表单解析失败 client_ip=%s error=%q", c.ClientIP(), err.Error()))
		c.String(http.StatusOK, "fail")
		return
	}

	values := c.Request.PostForm
	if len(values) == 0 {
		values = c.Request.URL.Query()
	}
	if len(values) == 0 {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝 webhook 参数为空 client_ip=%s", c.ClientIP()))
		c.String(http.StatusOK, "fail")
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("支付宝 webhook 参数 client_ip=%s trade_status=%q out_trade_no=%q app_id=%q sign_type=%q",
		c.ClientIP(),
		values.Get("trade_status"),
		values.Get("out_trade_no"),
		values.Get("app_id"),
		values.Get("sign_type"),
	))

	notification, err := client.DecodeNotification(ctx, values)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝 webhook 验签失败 client_ip=%s error=%q raw_app_id=%q trade_status=%q",
			c.ClientIP(), err.Error(), values.Get("app_id"), values.Get("trade_status")))
		c.String(http.StatusOK, "fail")
		return
	}

	if notification.AppId != strings.TrimSpace(setting.AlipayAppId) {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝 webhook app_id 不匹配 client_ip=%s notify_app_id=%q config_app_id=%q",
			c.ClientIP(), notification.AppId, setting.AlipayAppId))
		c.String(http.StatusOK, "fail")
		return
	}

	tradeNo := strings.TrimSpace(notification.OutTradeNo)
	if tradeNo == "" {
		logger.LogWarn(ctx, fmt.Sprintf("支付宝 webhook 缺少 out_trade_no client_ip=%s", c.ClientIP()))
		c.String(http.StatusOK, "fail")
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("支付宝 webhook 验签成功 trade_no=%s trade_status=%s client_ip=%s", tradeNo, notification.TradeStatus, c.ClientIP()))

	if notification.TradeStatus != alipay.TradeStatusSuccess && notification.TradeStatus != alipay.TradeStatusFinished {
		logger.LogInfo(ctx, fmt.Sprintf("支付宝 webhook 忽略非成功状态 trade_no=%s trade_status=%s", tradeNo, notification.TradeStatus))
		alipay.AckNotification(c.Writer)
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	payload := common.GetJsonString(notification)
	if err = model.CompleteSubscriptionOrder(tradeNo, payload, PaymentMethodAlipay, ""); err == nil {
		logger.LogInfo(ctx, fmt.Sprintf("支付宝 订阅订单完成 trade_no=%s", tradeNo))
		alipay.AckNotification(c.Writer)
		return
	} else if err != nil && !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
		logger.LogError(ctx, fmt.Sprintf("支付宝 订阅订单处理失败 trade_no=%s error=%q", tradeNo, err.Error()))
		c.String(http.StatusOK, "fail")
		return
	}

	if err = model.RechargeAlipay(tradeNo, c.ClientIP()); err != nil {
		logger.LogError(ctx, fmt.Sprintf("支付宝 充值处理失败 trade_no=%s error=%q", tradeNo, err.Error()))
		c.String(http.StatusOK, "fail")
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("支付宝 充值成功 trade_no=%s", tradeNo))
	alipay.AckNotification(c.Writer)
}

// AlipayCheckOrder actively queries Alipay for order status and completes the topup if paid.
// Called by the frontend after returning from the Alipay payment page.
func AlipayCheckOrder(c *gin.Context) {
ctx := c.Request.Context()
tradeNo := strings.TrimSpace(c.Query("out_trade_no"))
if tradeNo == "" {
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "缺少订单号"})
return
}

client, err := getAlipayClient()
if err != nil {
logger.LogError(ctx, fmt.Sprintf("支付宝 主动查单 获取客户端失败 trade_no=%s error=%q", tradeNo, err.Error()))
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "支付宝配置错误"})
return
}

rsp, err := client.TradeQuery(ctx, alipay.TradeQuery{OutTradeNo: tradeNo})
if err != nil {
logger.LogError(ctx, fmt.Sprintf("支付宝 主动查单 请求失败 trade_no=%s error=%q", tradeNo, err.Error()))
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "查询失败，请联系管理员"})
return
}

logger.LogInfo(ctx, fmt.Sprintf("支付宝 主动查单 trade_no=%s trade_status=%s", tradeNo, rsp.TradeStatus))

if rsp.TradeStatus != alipay.TradeStatusSuccess && rsp.TradeStatus != alipay.TradeStatusFinished {
c.JSON(http.StatusOK, gin.H{"message": "pending", "data": string(rsp.TradeStatus)})
return
}

LockOrder(tradeNo)
defer UnlockOrder(tradeNo)

if err = model.RechargeAlipay(tradeNo, c.ClientIP()); err != nil {
logger.LogError(ctx, fmt.Sprintf("支付宝 主动查单 充值处理失败 trade_no=%s error=%q", tradeNo, err.Error()))
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值处理失败"})
return
}

logger.LogInfo(ctx, fmt.Sprintf("支付宝 主动查单 充值成功 trade_no=%s", tradeNo))
c.JSON(http.StatusOK, gin.H{"message": "success", "data": "充值成功"})
}
