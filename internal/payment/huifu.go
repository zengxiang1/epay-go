// internal/payment/huifu.go
package payment

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const huifuAPIHost = "https://api.huifu.com"

// HuifuConfig 汇付天下（斗拱平台）配置
type HuifuConfig struct {
	SysID              string `json:"sys_id"`                   // 系统接入号
	ProductID          string `json:"product_id"`               // 产品号
	HuifuID            string `json:"huifu_id"`                 // 商户号
	MerchantPrivateKey string `json:"rsa_merchant_private_key"` // 商户私钥(PKCS8)
	HuifuPublicKey     string `json:"rsa_huifu_public_key"`     // 汇付公钥
}

// HuifuAdapter 汇付天下适配器
type HuifuAdapter struct {
	config     *HuifuConfig
	family     string // 承接方式: wechat / alipay，由注册的插件写死注入
	baseURL    string // 汇付 API host，默认 huifuAPIHost，测试可覆盖
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	httpClient *http.Client
}

// NewHuifuWechatAdapter 汇付-微信 适配器工厂（插件 hf-wxpay）
func NewHuifuWechatAdapter(configJSON json.RawMessage) (PaymentAdapter, error) {
	return newHuifuAdapter(configJSON, "wechat")
}

// NewHuifuAlipayAdapter 汇付-支付宝 适配器工厂（插件 hf-alipay）
func NewHuifuAlipayAdapter(configJSON json.RawMessage) (PaymentAdapter, error) {
	return newHuifuAdapter(configJSON, "alipay")
}

func newHuifuAdapter(configJSON json.RawMessage, family string) (PaymentAdapter, error) {
	var cfg HuifuConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, err
	}

	priv, err := parseHuifuPrivateKey(cfg.MerchantPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("解析商户私钥失败: %w", err)
	}
	pub, err := parseHuifuPublicKey(cfg.HuifuPublicKey)
	if err != nil {
		return nil, fmt.Errorf("解析汇付公钥失败: %w", err)
	}

	return &HuifuAdapter{
		config:     &cfg,
		family:     family,
		baseURL:    huifuAPIHost,
		privateKey: priv,
		publicKey:  pub,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func parseHuifuPrivateKey(key string) (*rsa.PrivateKey, error) {
	block := huifuPemBlock(key, "RSA PRIVATE KEY")
	if block == nil {
		return nil, errors.New("私钥格式错误")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("不是有效的RSA私钥")
	}
	return rsaKey, nil
}

func parseHuifuPublicKey(key string) (*rsa.PublicKey, error) {
	block := huifuPemBlock(key, "PUBLIC KEY")
	if block == nil {
		return nil, errors.New("公钥格式错误")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("不是有效的RSA公钥")
	}
	return rsaKey, nil
}

func huifuPemBlock(key, kind string) *pem.Block {
	k := strings.TrimSpace(key)
	if !strings.Contains(k, "BEGIN") {
		k = fmt.Sprintf("-----BEGIN %s-----\n%s\n-----END %s-----", kind, k, kind)
	}
	block, _ := pem.Decode([]byte(k))
	return block
}

func (h *HuifuAdapter) sign(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	sig, err := rsa.SignPKCS1v15(rand.Reader, h.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func (h *HuifuAdapter) verify(data []byte, sigBase64 string) error {
	sig, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(h.publicKey, crypto.SHA256, hash[:], sig)
}

// doRequest 按汇付协议构造请求信封、发送、验签并解析业务数据
func (h *HuifuAdapter) doRequest(ctx context.Context, path string, data map[string]interface{}, result interface{}) error {
	// data 字段用 map[string]interface{} 经 json.Marshal 序列化，Go 对 map 的 key 天然按字典序排序，
	// 满足汇付"加签前字段排序"的要求，无需手写排序逻辑。
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	sign, err := h.sign(dataJSON)
	if err != nil {
		return err
	}

	envelope := map[string]interface{}{
		"sys_id":     h.config.SysID,
		"product_id": h.config.ProductID,
		"sign":       sign,
		"data":       json.RawMessage(dataJSON),
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var envResp struct {
		Data json.RawMessage `json:"data"`
		Sign string          `json:"sign"`
	}
	if err := json.Unmarshal(respBody, &envResp); err != nil {
		return fmt.Errorf("解析汇付响应失败: %w, body=%s", err, string(respBody))
	}
	if err := h.verify(envResp.Data, envResp.Sign); err != nil {
		return fmt.Errorf("汇付响应验签失败: %w", err)
	}
	return json.Unmarshal(envResp.Data, result)
}

type huifuDataHeader struct {
	RespCode string `json:"resp_code"`
	RespDesc string `json:"resp_desc"`
}

func (h huifuDataHeader) isSuccess() bool {
	return h.RespCode == "00000000"
}

// resolveTradeType 按承接方式(family)+支付方式解析汇付的 trade_type
func (h *HuifuAdapter) resolveTradeType(payMethod string) (string, error) {
	isWechat := h.family == "wechat"
	switch payMethod {
	case "scan", "qrcode", "native", "precreate", "":
		if isWechat {
			return "T_NATIVE", nil
		}
		return "A_NATIVE", nil
	case "jsapi":
		if isWechat {
			return "T_JSAPI", nil
		}
		return "", errors.New("汇付支付宝承接方式暂不支持jsapi")
	case "h5", "wap":
		if isWechat {
			return "T_H5", nil
		}
		return "", errors.New("汇付支付宝承接方式暂不支持h5")
	default:
		return "", fmt.Errorf("unsupported pay method: %s", payMethod)
	}
}

// CreateOrder 创建支付订单
func (h *HuifuAdapter) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	tradeType, err := h.resolveTradeType(req.PayMethod)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"req_seq_id": req.TradeNo,
		"req_date":   time.Now().Format("20060102"),
		"huifu_id":   h.config.HuifuID,
		"goods_desc": req.Subject,
		"trade_type": tradeType,
		"trans_amt":  req.Amount.StringFixed(2),
		"notify_url": req.NotifyURL,
	}
	if tradeType == "T_JSAPI" {
		openid := req.Extra["openid"]
		if openid == "" {
			return nil, errors.New("jsapi pay requires openid")
		}
		wxData, _ := json.Marshal(map[string]string{"openid": openid})
		data["wx_data"] = string(wxData)
	}

	var result struct {
		huifuDataHeader
		QrCode         string `json:"qr_code"`
		PayInfo        string `json:"pay_info"`
		WxResponse     string `json:"wx_response"`
		AlipayResponse string `json:"alipay_response"`
	}
	if err := h.doRequest(ctx, "/v2/trade/payment/jspay", data, &result); err != nil {
		return nil, err
	}
	if !result.isSuccess() {
		return nil, errors.New(result.RespDesc)
	}

	switch {
	case result.QrCode != "":
		return &CreateOrderResponse{PayType: "qrcode", PayURL: result.QrCode}, nil
	case result.PayInfo != "":
		return &CreateOrderResponse{PayType: "jsapi", PayParams: result.PayInfo}, nil
	case tradeType == "T_H5":
		if u := huifuExtractURL(result.WxResponse); u != "" {
			return &CreateOrderResponse{PayType: "redirect", PayURL: u}, nil
		}
		return nil, errors.New("无法从汇付H5响应中解析支付跳转地址")
	default:
		return nil, errors.New("汇付未返回可用的支付参数")
	}
}

// huifuExtractURL 尽力从汇付H5响应的原始JSON字符串中解析跳转地址
func huifuExtractURL(raw string) string {
	if raw == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	for _, key := range []string{"mweb_url", "h5_url", "url"} {
		if v, ok := m[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// QueryOrder 查询订单
func (h *HuifuAdapter) QueryOrder(ctx context.Context, tradeNo string) (*QueryOrderResponse, error) {
	orgReqDate := ""
	if len(tradeNo) >= 8 {
		orgReqDate = tradeNo[:8]
	}

	data := map[string]interface{}{
		"huifu_id":       h.config.HuifuID,
		"org_req_date":   orgReqDate,
		"org_req_seq_id": tradeNo,
	}
	var result struct {
		huifuDataHeader
		OutTransId string `json:"out_trans_id"`
		TransAmt   string `json:"trans_amt"`
		TransStat  string `json:"trans_stat"`
		EndTime    string `json:"end_time"`
	}
	if err := h.doRequest(ctx, "/v2/trade/payment/scanpay/query", data, &result); err != nil {
		return nil, err
	}
	if !result.isSuccess() {
		return nil, errors.New(result.RespDesc)
	}

	status := "pending"
	switch result.TransStat {
	case "S":
		status = "paid"
	case "F":
		status = "closed"
	}

	amount, _ := decimal.NewFromString(result.TransAmt)
	return &QueryOrderResponse{
		TradeNo:    tradeNo,
		ApiTradeNo: result.OutTransId,
		Amount:     amount,
		Status:     status,
		PaidAt:     result.EndTime,
	}, nil
}

// Refund 退款
func (h *HuifuAdapter) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	orgReqDate := ""
	if len(req.TradeNo) >= 8 {
		orgReqDate = req.TradeNo[:8]
	}

	data := map[string]interface{}{
		"req_date":   time.Now().Format("20060102"),
		"req_seq_id": req.RefundNo,
		"huifu_id":   h.config.HuifuID,
		// 汇付 ord_amt 表示【本次退款金额】，不是原订单总额；传错会导致按原订单全额退款
		"ord_amt":        req.Amount.StringFixed(2),
		"org_req_date":   orgReqDate,
		"org_req_seq_id": req.TradeNo,
		"remark":         req.RefundDesc,
	}
	var result struct {
		huifuDataHeader
		TransStat string `json:"trans_stat"`
	}
	if err := h.doRequest(ctx, "/v2/trade/payment/scanpay/refund", data, &result); err != nil {
		return nil, err
	}
	if !result.isSuccess() {
		return &RefundResponse{RefundNo: req.RefundNo, Status: "failed", ErrorMessage: result.RespDesc}, nil
	}

	status := "processing"
	switch result.TransStat {
	case "S":
		status = "success"
	case "F":
		status = "failed"
	}

	return &RefundResponse{RefundNo: req.RefundNo, Status: status}, nil
}

// ParseNotify 解析回调通知
func (h *HuifuAdapter) ParseNotify(ctx context.Context, r *http.Request) (*NotifyResult, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}

	respData := values.Get("resp_data")
	sign := values.Get("sign")
	if respData == "" || sign == "" {
		return nil, errors.New("回调缺少 resp_data 或 sign")
	}
	if err := h.verify([]byte(respData), sign); err != nil {
		return nil, fmt.Errorf("回调验签失败: %w", err)
	}

	var n struct {
		huifuDataHeader
		ReqSeqId  string `json:"req_seq_id"`
		HfSeqId   string `json:"hf_seq_id"`
		TransAmt  string `json:"trans_amt"`
		TransStat string `json:"trans_stat"`
	}
	if err := json.Unmarshal([]byte(respData), &n); err != nil {
		return nil, err
	}

	status := "fail"
	if n.TransStat == "S" {
		status = "success"
	}

	amount, _ := decimal.NewFromString(n.TransAmt)
	return &NotifyResult{
		TradeNo:    n.ReqSeqId,
		ApiTradeNo: n.HfSeqId,
		Amount:     amount,
		Status:     status,
	}, nil
}

// NotifySuccess 返回回调成功响应
func (h *HuifuAdapter) NotifySuccess() string {
	return "success"
}

func init() {
	Register("hf-wxpay", NewHuifuWechatAdapter)
	Register("hf-alipay", NewHuifuAlipayAdapter)
}
