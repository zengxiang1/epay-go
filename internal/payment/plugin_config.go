// internal/payment/plugin_config.go
package payment

// PluginConfigField 配置字段定义
type PluginConfigField struct {
	Key         string            `json:"key"`         // 字段键名
	Name        string            `json:"name"`        // 显示名称
	Type        string            `json:"type"`        // 类型: input/textarea/select/checkbox
	Required    bool              `json:"required"`    // 是否必填
	Placeholder string            `json:"placeholder"` // 占位符
	Note        string            `json:"note"`        // 说明文字
	Options     map[string]string `json:"options"`     // 下拉选项（type=select时使用）
}

// PluginConfig 插件配置信息
type PluginConfig struct {
	Name     string              `json:"name"`      // 插件英文名
	ShowName string              `json:"show_name"` // 显示名称
	Author   string              `json:"author"`    // 作者
	Link     string              `json:"link"`      // 官方链接
	Inputs   []PluginConfigField `json:"inputs"`    // 配置字段
	PayTypes []PayTypeOption     `json:"pay_types"` // 支持的支付接口
	BindWxmp bool                `json:"bind_wxmp"` // 是否绑定微信公众号
	BindWxa  bool                `json:"bind_wxa"`  // 是否绑定微信小程序
	Note     string              `json:"note"`      // 配置说明
}

// PayTypeOption 支付接口选项
type PayTypeOption struct {
	Code string `json:"code"` // 接口代码
	Name string `json:"name"` // 接口名称
}

// GetPluginConfigs 获取所有插件配置模板
func GetPluginConfigs() map[string]PluginConfig {
	return map[string]PluginConfig{
		"alipay":    GetAlipayConfig(),
		"wechat":    GetWechatConfig(),
		"hf-wxpay":  GetHuifuWechatConfig(),
		"hf-alipay": GetHuifuAlipayConfig(),
	}
}

// GetAlipayConfig 支付宝配置模板
func GetAlipayConfig() PluginConfig {
	return PluginConfig{
		Name:     "alipay",
		ShowName: "支付宝官方支付",
		Author:   "支付宝",
		Link:     "https://open.alipay.com",
		Inputs: []PluginConfigField{
			{
				Key:         "app_id",
				Name:        "应用APPID",
				Type:        "input",
				Required:    true,
				Placeholder: "请输入支付宝开放平台应用ID",
			},
			{
				Key:         "private_key",
				Name:        "应用私钥（RSA2）",
				Type:        "textarea",
				Required:    true,
				Placeholder: "请输入RSA2私钥（PKCS1或PKCS8格式）",
				Note:        "用于对请求参数进行签名",
			},
			{
				Key:         "public_key",
				Name:        "支付宝公钥",
				Type:        "textarea",
				Required:    true,
				Placeholder: "请输入支付宝公钥",
				Note:        "用于验证回调签名（如使用证书模式，可填支付宝公钥证书内容/或保持与后端策略一致）",
			},
			{
				Key:      "is_prod",
				Name:     "是否生产环境",
				Type:     "select",
				Required: true,
				Options: map[string]string{
					"false": "否（沙箱/测试）",
					"true":  "是（生产）",
				},
				Note: "沙箱请选 false；生产请选择 true",
			},
			{
				Key:      "sign_type",
				Name:     "签名类型",
				Type:     "select",
				Required: true,
				Options: map[string]string{
					"RSA2": "RSA2",
				},
				Note: "支付宝接口默认使用 RSA2",
			},
		},
		PayTypes: []PayTypeOption{
			{Code: "page", Name: "电脑网站支付"},
			{Code: "wap", Name: "手机网站支付"},
			{Code: "qrcode", Name: "当面付扫码"},
			{Code: "jsapi", Name: "当面付JS"},
			{Code: "app", Name: "APP支付"},
		},
		Note: "选择可用的支付接口，只能选择已经签约的产品。",
	}
}

// GetWechatConfig 微信支付配置模板
func GetWechatConfig() PluginConfig {
	return PluginConfig{
		Name:     "wechat",
		ShowName: "微信官方支付",
		Author:   "微信支付",
		Link:     "https://pay.weixin.qq.com",
		Inputs: []PluginConfigField{
			{
				Key:         "app_id",
				Name:        "公众号/小程序/开放平台AppID",
				Type:        "input",
				Required:    true,
				Placeholder: "请输入微信应用标识",
			},
			{
				Key:         "mch_id",
				Name:        "商户号",
				Type:        "input",
				Required:    true,
				Placeholder: "请输入微信支付商户号",
			},
			{
				Key:         "api_v3_key",
				Name:        "APIv3密钥",
				Type:        "input",
				Required:    true,
				Placeholder: "请输入APIv3密钥（32位）",
				Note:        "微信支付平台设置的 APIv3 Key",
			},
			{
				Key:         "serial_no",
				Name:        "商户证书序列号",
				Type:        "input",
				Required:    true,
				Placeholder: "请输入商户API证书序列号",
				Note:        "位于商户API证书详情页",
			},
			{
				Key:         "private_key",
				Name:        "商户私钥内容",
				Type:        "textarea",
				Required:    true,
				Placeholder: "请粘贴商户私钥 PEM 内容（含 BEGIN/END 行）",
				Note:        "用于微信支付 V3 签名，请勿上传证书文件本身",
			},
			{
				Key:         "platform_serial_no",
				Name:        "平台证书序列号（可选）",
				Type:        "input",
				Required:    false,
				Placeholder: "可留空（自动下载平台证书的场景需配合实现）",
			},
			{
				Key:         "platform_cert_content",
				Name:        "平台证书内容（可选）",
				Type:        "textarea",
				Required:    false,
				Placeholder: "可留空；如你手动配置平台证书，请粘贴 PEM 内容",
			},
		},
		PayTypes: []PayTypeOption{
			{Code: "native", Name: "扫码支付"},
			{Code: "jsapi", Name: "公众号支付"},
			{Code: "h5", Name: "H5支付"},
			{Code: "miniapp", Name: "小程序支付"},
			{Code: "app", Name: "APP支付"},
		},
		BindWxmp: true,
		BindWxa:  true,
		Note:     "当前后端使用微信支付 V3：必须配置 APIv3密钥、商户证书序列号、商户私钥内容。",
	}
}

// huifuCommonInputs 汇付两个插件共用的商户凭证字段
func huifuCommonInputs() []PluginConfigField {
	return []PluginConfigField{
		{
			Key:         "sys_id",
			Name:        "系统接入号",
			Type:        "input",
			Required:    true,
			Placeholder: "请输入汇付分配的系统接入号(system_id)",
		},
		{
			Key:         "product_id",
			Name:        "产品号",
			Type:        "input",
			Required:    true,
			Placeholder: "请输入产品号(product_id)",
		},
		{
			Key:         "huifu_id",
			Name:        "商户号",
			Type:        "input",
			Required:    true,
			Placeholder: "请输入汇付商户号(huifu_id)",
		},
		{
			Key:         "rsa_merchant_private_key",
			Name:        "商户私钥",
			Type:        "textarea",
			Required:    true,
			Placeholder: "请粘贴商户RSA私钥(PKCS8格式) PEM 内容",
			Note:        "用于对请求参数进行 RSA-SHA256 签名",
		},
		{
			Key:         "rsa_huifu_public_key",
			Name:        "汇付公钥",
			Type:        "textarea",
			Required:    true,
			Placeholder: "请粘贴汇付平台公钥 PEM 内容",
			Note:        "用于验证汇付返回及异步通知的签名",
		},
	}
}

// GetHuifuWechatConfig 汇付天下-微信 配置模板（插件 hf-wxpay）
func GetHuifuWechatConfig() PluginConfig {
	return PluginConfig{
		Name:     "hf-wxpay",
		ShowName: "汇付天下（微信）",
		Author:   "汇付天下",
		Link:     "https://paas.huifu.com",
		Inputs:   huifuCommonInputs(),
		PayTypes: []PayTypeOption{
			{Code: "scan", Name: "扫码支付"},
			{Code: "jsapi", Name: "JS调起支付"},
			{Code: "h5", Name: "H5支付"},
		},
		Note: "汇付-微信承接：用于处理微信支付，客户端下单传 type=wxpay 即可路由到本通道。",
	}
}

// GetHuifuAlipayConfig 汇付天下-支付宝 配置模板（插件 hf-alipay）
func GetHuifuAlipayConfig() PluginConfig {
	return PluginConfig{
		Name:     "hf-alipay",
		ShowName: "汇付天下（支付宝）",
		Author:   "汇付天下",
		Link:     "https://paas.huifu.com",
		Inputs:   huifuCommonInputs(),
		PayTypes: []PayTypeOption{
			{Code: "scan", Name: "扫码支付"},
		},
		Note: "汇付-支付宝承接：用于处理支付宝支付，客户端下单传 type=alipay 即可路由到本通道；当前仅支持扫码。",
	}
}
