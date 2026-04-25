package controller

import (
"bytes"
"crypto"
"crypto/aes"
"crypto/cipher"
"crypto/rand"
"crypto/rsa"
"crypto/sha256"
"crypto/x509"
"encoding/base64"
"encoding/pem"
"fmt"
"io"
"net/http"
"strings"
"time"

"github.com/QuantumNous/new-api/common"
"github.com/QuantumNous/new-api/logger"
"github.com/QuantumNous/new-api/model"
"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
"github.com/QuantumNous/new-api/setting/operation_setting"
"github.com/gin-gonic/gin"
"github.com/shopspring/decimal"
)

const (
PaymentMethodWechatNative  = "wechat_native"
wechatPayHost              = "https://api.mch.weixin.qq.com"
wechatNativeOrderPath      = "/v3/pay/transactions/native"
wechatQueryOrderPathPrefix = "/v3/pay/transactions/out-trade-no/"
)

// ----------- config check -----------

func isWechatNativePayEnabled() bool {
return setting.WechatPayEnabled &&
strings.TrimSpace(setting.WechatPayAppId) != "" &&
strings.TrimSpace(setting.WechatPayMchId) != "" &&
strings.TrimSpace(setting.WechatPayApiV3Key) != "" &&
strings.TrimSpace(setting.WechatPayPrivateKey) != "" &&
strings.TrimSpace(setting.WechatPayCertSerialNo) != "" &&
strings.TrimSpace(setting.WechatPayPublicKeyId) != "" &&
strings.TrimSpace(setting.WechatPayPublicKey) != ""
}

// ----------- crypto helpers -----------

func loadRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
pemStr = strings.TrimSpace(pemStr)
pemStr = strings.ReplaceAll(pemStr, "\\n", "\n")
block, _ := pem.Decode([]byte(pemStr))
if block == nil {
return nil, fmt.Errorf("微信支付：私钥 PEM 解析失败")
}
if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
if rsaKey, ok := key.(*rsa.PrivateKey); ok {
return rsaKey, nil
}
return nil, fmt.Errorf("微信支付：私钥类型不是 RSA")
}
return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func loadRSAPublicKey(pemStr string) (*rsa.PublicKey, error) {
pemStr = strings.TrimSpace(pemStr)
pemStr = strings.ReplaceAll(pemStr, "\\n", "\n")
block, _ := pem.Decode([]byte(pemStr))
if block == nil {
return nil, fmt.Errorf("微信支付：公钥 PEM 解析失败")
}
pub, err := x509.ParsePKIXPublicKey(block.Bytes)
if err != nil {
return nil, err
}
rsaKey, ok := pub.(*rsa.PublicKey)
if !ok {
return nil, fmt.Errorf("微信支付：公钥类型不是 RSA")
}
return rsaKey, nil
}

func buildWechatNonce() string {
b := make([]byte, 16)
_, _ = rand.Read(b)
return fmt.Sprintf("%x", b)
}

// buildWechatAuthorization 构造 WECHATPAY2-SHA256-RSA2048 Authorization 头
func buildWechatAuthorization(method, urlPath, body string) (string, error) {
timestamp := fmt.Sprintf("%d", time.Now().Unix())
nonce := buildWechatNonce()
message := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", method, urlPath, timestamp, nonce, body)

privateKey, err := loadRSAPrivateKey(setting.WechatPayPrivateKey)
if err != nil {
return "", err
}
h := sha256.New()
h.Write([]byte(message))
digest := h.Sum(nil)
sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest)
if err != nil {
return "", err
}
sigB64 := base64.StdEncoding.EncodeToString(sig)

auth := fmt.Sprintf(
`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",timestamp="%s",serial_no="%s",signature="%s"`,
strings.TrimSpace(setting.WechatPayMchId),
nonce,
timestamp,
strings.TrimSpace(setting.WechatPayCertSerialNo),
sigB64,
)
return auth, nil
}

// verifyWechatCallback 用微信支付公钥验证回调签名
func verifyWechatCallback(timestamp, nonce, body, signatureB64 string) error {
message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, body)
sig, err := base64.StdEncoding.DecodeString(signatureB64)
if err != nil {
return fmt.Errorf("base64 解码签名失败: %w", err)
}
pub, err := loadRSAPublicKey(setting.WechatPayPublicKey)
if err != nil {
return err
}
h := sha256.New()
h.Write([]byte(message))
digest := h.Sum(nil)
return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest, sig)
}

// decryptWechatResource 使用 APIv3Key 解密 AEAD_AES_256_GCM 密文
func decryptWechatResource(ciphertext, nonce, associatedData string) ([]byte, error) {
key := []byte(strings.TrimSpace(setting.WechatPayApiV3Key))
if len(key) != 32 {
return nil, fmt.Errorf("微信支付 APIv3Key 长度必须为 32 字节，当前: %d", len(key))
}
ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
if err != nil {
return nil, fmt.Errorf("密文 base64 解码失败: %w", err)
}
block, err := aes.NewCipher(key)
if err != nil {
return nil, err
}
gcm, err := cipher.NewGCM(block)
if err != nil {
return nil, err
}
plaintext, err := gcm.Open(nil, []byte(nonce), ciphertextBytes, []byte(associatedData))
if err != nil {
return nil, fmt.Errorf("AES-GCM 解密失败: %w", err)
}
return plaintext, nil
}

// ----------- WeChat API structs -----------

type wechatAmountInfo struct {
Total    int64  `json:"total"`
Currency string `json:"currency"`
}

type wechatNativeOrderRequest struct {
AppId       string           `json:"appid"`
MchId       string           `json:"mchid"`
Description string           `json:"description"`
OutTradeNo  string           `json:"out_trade_no"`
TimeExpire  string           `json:"time_expire"`
NotifyUrl   string           `json:"notify_url"`
Amount      wechatAmountInfo `json:"amount"`
}

type wechatNativeOrderResponse struct {
CodeURL string `json:"code_url"`
}

type wechatCallbackResource struct {
Algorithm      string `json:"algorithm"`
Ciphertext     string `json:"ciphertext"`
AssociatedData string `json:"associated_data"`
Nonce          string `json:"nonce"`
OriginalType   string `json:"original_type"`
}

type wechatCallbackBody struct {
Id           string                 `json:"id"`
EventType    string                 `json:"event_type"`
ResourceType string                 `json:"resource_type"`
Resource     wechatCallbackResource `json:"resource"`
}

type wechatPaymentTransaction struct {
TradeState string `json:"trade_state"`
OutTradeNo string `json:"out_trade_no"`
TransactionId string `json:"transaction_id"`
}

// ----------- API calls -----------

func doWechatRequest(method, urlPath, body string) ([]byte, int, error) {
auth, err := buildWechatAuthorization(method, urlPath, body)
if err != nil {
return nil, 0, err
}

var reqBody io.Reader
if body != "" {
reqBody = bytes.NewBufferString(body)
}
req, err := http.NewRequest(method, wechatPayHost+urlPath, reqBody)
if err != nil {
return nil, 0, err
}
req.Header.Set("Authorization", auth)
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Accept", "application/json")

client := &http.Client{Timeout: 15 * time.Second}
resp, err := client.Do(req)
if err != nil {
return nil, 0, err
}
defer resp.Body.Close()
respBody, err := io.ReadAll(resp.Body)
return respBody, resp.StatusCode, err
}

func callWechatNativeOrder(orderReq wechatNativeOrderRequest) (string, error) {
bodyBytes, err := common.Marshal(orderReq)
if err != nil {
return "", err
}
body := string(bodyBytes)

respBody, statusCode, err := doWechatRequest(http.MethodPost, wechatNativeOrderPath, body)
if err != nil {
return "", fmt.Errorf("微信 Native 下单请求失败: %w", err)
}
if statusCode != http.StatusOK {
return "", fmt.Errorf("微信 Native 下单失败，HTTP %d: %s", statusCode, safePrefix(string(respBody), 200))
}

var resp wechatNativeOrderResponse
if err := common.Unmarshal(respBody, &resp); err != nil {
return "", fmt.Errorf("微信 Native 下单响应解析失败: %w", err)
}
if resp.CodeURL == "" {
return "", fmt.Errorf("微信 Native 下单 code_url 为空: %s", safePrefix(string(respBody), 200))
}
return resp.CodeURL, nil
}

func callWechatQueryOrder(outTradeNo string) (*wechatPaymentTransaction, error) {
path := fmt.Sprintf("%s%s?mchid=%s", wechatQueryOrderPathPrefix, outTradeNo, strings.TrimSpace(setting.WechatPayMchId))
respBody, statusCode, err := doWechatRequest(http.MethodGet, path, "")
if err != nil {
return nil, fmt.Errorf("微信查单请求失败: %w", err)
}
if statusCode != http.StatusOK {
return nil, fmt.Errorf("微信查单失败，HTTP %d: %s", statusCode, safePrefix(string(respBody), 200))
}
var tx wechatPaymentTransaction
if err := common.Unmarshal(respBody, &tx); err != nil {
return nil, fmt.Errorf("微信查单响应解析失败: %w", err)
}
return &tx, nil
}

func safePrefix(s string, n int) string {
if len(s) <= n {
return s
}
return s[:n]
}

// ----------- Controller handlers -----------

// RequestWechatNativePay handles WeChat Native Pay topup creation and returns true when request is handled.
func RequestWechatNativePay(c *gin.Context, req *EpayRequest, payMoney float64) bool {
if req.PaymentMethod != PaymentMethodWechatNative || !isWechatNativePayEnabled() {
return false
}
if payMoney < 0.01 {
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
return true
}

id := c.GetInt("id")
tradeNo := fmt.Sprintf("WXUSR%dNO%s%d", id, common.GetRandomString(6), time.Now().Unix())

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
PaymentMethod: PaymentMethodWechatNative,
CreateTime:    time.Now().Unix(),
Status:        common.TopUpStatusPending,
}
if err := topUp.Insert(); err != nil {
logger.LogError(c.Request.Context(), fmt.Sprintf("微信 创建订单失败 error=%q", err.Error()))
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
return true
}

// 到期时间 15 分钟后，RFC3339 格式
timeExpire := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)

	callbackBase := getWechatNotifyURL()

orderReq := wechatNativeOrderRequest{
AppId:       strings.TrimSpace(setting.WechatPayAppId),
MchId:       strings.TrimSpace(setting.WechatPayMchId),
Description: fmt.Sprintf("TUC%d", req.Amount),
OutTradeNo:  tradeNo,
TimeExpire:  timeExpire,
NotifyUrl:   callbackBase,
Amount: wechatAmountInfo{
Total:    int64(payMoney * 100), // 分
Currency: "CNY",
},
}

codeURL, err := callWechatNativeOrder(orderReq)
if err != nil {
logger.LogError(c.Request.Context(), fmt.Sprintf("微信 Native 下单失败 trade_no=%s error=%q", tradeNo, err.Error()))
topUp.Status = common.TopUpStatusFailed
_ = topUp.Update()
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "下单失败，请联系管理员"})
return true
}

logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信 Native 下单成功 trade_no=%s", tradeNo))
c.JSON(http.StatusOK, gin.H{
"message": "success",
"data": gin.H{
"code_url": codeURL,
"order_id": tradeNo,
},
})
return true
}

func getWechatNotifyURL() string {
	return strings.TrimRight(service.GetCallbackAddress(), "/") + "/api/wechat/notify"
}

// WechatPayNotify handles WeChat Pay payment callbacks.
func WechatPayNotify(c *gin.Context) {
ctx := c.Request.Context()
logger.LogInfo(ctx, fmt.Sprintf("微信 webhook 收到请求 client_ip=%s", c.ClientIP()))

bodyBytes, err := io.ReadAll(c.Request.Body)
if err != nil {
logger.LogError(ctx, fmt.Sprintf("微信 webhook 读取 body 失败 error=%q", err.Error()))
c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "读取请求体失败"})
return
}
body := string(bodyBytes)

timestamp := c.GetHeader("Wechatpay-Timestamp")
nonce := c.GetHeader("Wechatpay-Nonce")
signature := c.GetHeader("Wechatpay-Signature")

if timestamp == "" || nonce == "" || signature == "" {
logger.LogWarn(ctx, "微信 webhook 缺少签名头")
c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "缺少签名头"})
return
}

if err := verifyWechatCallback(timestamp, nonce, body, signature); err != nil {
logger.LogError(ctx, fmt.Sprintf("微信 webhook 验签失败 client_ip=%s error=%q", c.ClientIP(), err.Error()))
c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "签名验证失败"})
return
}

var cb wechatCallbackBody
if err := common.Unmarshal(bodyBytes, &cb); err != nil {
logger.LogError(ctx, fmt.Sprintf("微信 webhook 解析 JSON 失败 error=%q", err.Error()))
c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "解析请求体失败"})
return
}

if cb.EventType != "TRANSACTION.SUCCESS" {
logger.LogInfo(ctx, fmt.Sprintf("微信 webhook 忽略事件类型 event_type=%s", cb.EventType))
c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
return
}

plaintext, err := decryptWechatResource(cb.Resource.Ciphertext, cb.Resource.Nonce, cb.Resource.AssociatedData)
if err != nil {
logger.LogError(ctx, fmt.Sprintf("微信 webhook 解密失败 error=%q", err.Error()))
c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "解密失败"})
return
}

var tx wechatPaymentTransaction
if err := common.Unmarshal(plaintext, &tx); err != nil {
logger.LogError(ctx, fmt.Sprintf("微信 webhook 解析交易信息失败 error=%q", err.Error()))
c.JSON(http.StatusBadRequest, gin.H{"code": "FAIL", "message": "解析交易信息失败"})
return
}

if tx.TradeState != "SUCCESS" {
logger.LogInfo(ctx, fmt.Sprintf("微信 webhook 交易状态非 SUCCESS trade_no=%s trade_state=%s", tx.OutTradeNo, tx.TradeState))
c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
return
}

tradeNo := tx.OutTradeNo
logger.LogInfo(ctx, fmt.Sprintf("微信 webhook 验签成功 trade_no=%s", tradeNo))

LockOrder(tradeNo)
defer UnlockOrder(tradeNo)

if err := model.RechargeWechat(tradeNo, c.ClientIP()); err != nil {
logger.LogError(ctx, fmt.Sprintf("微信 webhook 充值处理失败 trade_no=%s error=%q", tradeNo, err.Error()))
c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": "充值处理失败"})
return
}

logger.LogInfo(ctx, fmt.Sprintf("微信 webhook 充值成功 trade_no=%s", tradeNo))
c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "OK"})
}

// WechatPayCheckOrder actively queries WeChat for order status.
func WechatPayCheckOrder(c *gin.Context) {
ctx := c.Request.Context()
tradeNo := strings.TrimSpace(c.Query("out_trade_no"))
if tradeNo == "" {
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "缺少订单号"})
return
}

tx, err := callWechatQueryOrder(tradeNo)
if err != nil {
logger.LogError(ctx, fmt.Sprintf("微信 主动查单 失败 trade_no=%s error=%q", tradeNo, err.Error()))
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "查询失败，请联系管理员"})
return
}

logger.LogInfo(ctx, fmt.Sprintf("微信 主动查单 trade_no=%s trade_state=%s", tradeNo, tx.TradeState))

if tx.TradeState != "SUCCESS" {
c.JSON(http.StatusOK, gin.H{"message": "pending", "data": tx.TradeState})
return
}

LockOrder(tradeNo)
defer UnlockOrder(tradeNo)

if err := model.RechargeWechat(tradeNo, c.ClientIP()); err != nil {
logger.LogError(ctx, fmt.Sprintf("微信 主动查单 充值处理失败 trade_no=%s error=%q", tradeNo, err.Error()))
c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值处理失败"})
return
}

logger.LogInfo(ctx, fmt.Sprintf("微信 主动查单 充值成功 trade_no=%s", tradeNo))
c.JSON(http.StatusOK, gin.H{"message": "success", "data": "充值成功"})
}
