package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
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

	payLink, err := requestAlipayPagePayLink(
		fmt.Sprintf("TUC%d", req.Amount),
		tradeNo,
		payMoney,
		getAlipayTopupReturnURL(),
	)
	if err != nil {
		log.Printf("拉起支付宝支付失败: %v", err)
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

	payLink, err := requestAlipayPagePayLink(
		fmt.Sprintf("SUB:%s", plan.Title),
		tradeNo,
		plan.PriceAmount,
		getAlipaySubscriptionReturnURL(),
	)
	if err != nil {
		log.Printf("拉起支付宝订阅支付失败: %v", err)
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
	client, err := getAlipayClient()
	if err != nil {
		log.Printf("支付宝回调配置错误: %v", err)
		c.String(http.StatusOK, "fail")
		return
	}

	if err = c.Request.ParseForm(); err != nil {
		log.Printf("解析支付宝回调参数失败: %v", err)
		c.String(http.StatusOK, "fail")
		return
	}

	values := c.Request.PostForm
	if len(values) == 0 {
		values = c.Request.URL.Query()
	}
	if len(values) == 0 {
		log.Printf("支付宝回调参数为空")
		c.String(http.StatusOK, "fail")
		return
	}

	notification, err := client.DecodeNotification(c.Request.Context(), values)
	if err != nil {
		log.Printf("支付宝回调验签失败: %v", err)
		c.String(http.StatusOK, "fail")
		return
	}

	if notification.AppId != strings.TrimSpace(setting.AlipayAppId) {
		log.Printf("支付宝回调 app_id 不匹配: %s", notification.AppId)
		c.String(http.StatusOK, "fail")
		return
	}

	tradeNo := strings.TrimSpace(notification.OutTradeNo)
	if tradeNo == "" {
		log.Printf("支付宝回调缺少 out_trade_no")
		c.String(http.StatusOK, "fail")
		return
	}

	if notification.TradeStatus != alipay.TradeStatusSuccess && notification.TradeStatus != alipay.TradeStatusFinished {
		alipay.AckNotification(c.Writer)
		return
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	payload := common.GetJsonString(notification)
	if err = model.CompleteSubscriptionOrder(tradeNo, payload, PaymentMethodAlipay); err == nil {
		alipay.AckNotification(c.Writer)
		return
	} else if err != nil && !errors.Is(err, model.ErrSubscriptionOrderNotFound) {
		log.Printf("支付宝订阅订单处理失败: %v, 订单号: %s", err, tradeNo)
		c.String(http.StatusOK, "fail")
		return
	}

	if err = model.RechargeAlipay(tradeNo); err != nil {
		log.Printf("支付宝充值处理失败: %v, 订单号: %s", err, tradeNo)
		c.String(http.StatusOK, "fail")
		return
	}

	alipay.AckNotification(c.Writer)
}
