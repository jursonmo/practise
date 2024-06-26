package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/go-pay/gopay/pkg/xlog"
)

//沙箱，支付账号ilqohu3459@sandbox.com， 密码111111
//https://open.alipay.com/develop/sandbox/account

var appId = "9021000122692113"

// 复制“应用公钥”至支付宝开放平台，进而获取支付宝公钥。
var appPubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAl5rsu2H1Vby3INlEWaQSgDSOSomTppp4b/lgEgGK6tVeDfBZmSRxmX5PpweSPrf/szFkM0cUbXyrhLrIH37uLEquAIQokXCXjQBSazee2edRVotwEDY/th77thlvUTf/9A74+WNVDyiVcgUyN2G+PZtrphPXFQ1GXLY5EhrvUoRuOdozSXHzbQpTS4k0q/YssqSjQSccjto9A3wsMDMXLHqYDck2Ra1gXFy2Yjjg5KowEoJJYjY8aC78EG8shInkEEoLiFHR1F0mAcoXDAAegvgECYHDXtdzjStsgcLOnZ1ogeAPwMR88fRcX3ioanVYKotV/4130+JeQ2QQbCRWkwIDAQAB"
var appPrivateKey = "MIIEogIBAAKCAQEAl5rsu2H1Vby3INlEWaQSgDSOSomTppp4b/lgEgGK6tVeDfBZmSRxmX5PpweSPrf/szFkM0cUbXyrhLrIH37uLEquAIQokXCXjQBSazee2edRVotwEDY/th77thlvUTf/9A74+WNVDyiVcgUyN2G+PZtrphPXFQ1GXLY5EhrvUoRuOdozSXHzbQpTS4k0q/YssqSjQSccjto9A3wsMDMXLHqYDck2Ra1gXFy2Yjjg5KowEoJJYjY8aC78EG8shInkEEoLiFHR1F0mAcoXDAAegvgECYHDXtdzjStsgcLOnZ1ogeAPwMR88fRcX3ioanVYKotV/4130+JeQ2QQbCRWkwIDAQABAoIBAGgYWSnEdhbbkAY/CF5geM+MxpLJahdAAygnW16hrofV31HE0VCEpHeXMgvm9/SWlDyu0jUfPhh7PK3TLivqFJFW6aizFcPfQj/vk6fItgq6eK/q6BRJm29qULzVNAjZYaZrTWq3WKUi4ZI7nSJHu79DYyPShaEPz1tDR2Z2FhNahwzWP+DQV2MIBn3RGValzSm7gI/RVp00YB0CFi9+eG/aYNoaV1hxp++8Q0CjuThg0B20bWel5Tu5McxRJL7vYeJrkQZPWxpkAEr0nyaLcLQYK11RzLcK3Twu6w69mPK4qbQLqnz9YYmqlcbxiHxy5u7ZitllaXIRDwHlHIlf1GkCgYEA5ox1kdTngCL9n+oIkEB37SMpFblAqhpaPN6FrWaOVS+q8dftidd3shx66KTwfhpnWY4P88t7gNSM0PJko/ES1c0dZmvzJGwKWOAIrl9Ym8OorLe2yL4uq5CBQBRJmXTic33yot38toAbry/mSPPF+6pBtUycPAHRryr8+DtoLcUCgYEAqFdxFedAvJQnAmnfldoNfMVwbGhYAj3p+UaLQwy5h1xDMeDv48e4fGxEQIzKK5Jpbe7p18Uk9uGThCiuEFr2BjDY9nP79Bs5ri8Rey4KvLQ2HQbuIZklgW54/ZpxAAttq130OfLIyOnZi14IPb0+YBS3jXb7+yr51iI0Rxrb0HcCgYBZmIp0QxY4gOCpzezICpXQrZJg496SfK1G7H9s1OdJib3YQL8Ki5bzvAez862WhDJX5lKivxhfB7s11I1x/NUCC7V6QUd7hxU6Vs5o3Zr05cPeY4MAXpCKkhz4xymXHoqGsZKi4rw8PGsF2QqYnUv6sr7Yc05gL6DKf11SJtwktQKBgGt2Rm5hDWZUfQKBa4VRiUKZF0dc5LGprG7Apa3LtbO/JfX1Ta3ulMp4oqlCNtzRvhO7a/OdmhcvsOewwE0Yg+03yYiqSbBuoMecrGAh6CDGObUV83XnOZYCW6IosPICWaQHehxz69C113WsNT6US/kxwGrCBeE0cgBMHWs2rhPJAoGAVmUHc8j2RroiFYJbCQBhtCPTsmq0iV/a6y5E+c51c8mnAmPffn2uXpjdc2kweU9TJ6zkllL+9x9Yn0kowRO7T8cDg0CEOje1VWqwx1MMP9lkh/viSZ6ah5qOd54OL6BrjhNWZ50nrNeynmXE/aqEsJRHN8qBx63uRb8UdN1Jiws="

var aliPayPublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAmXRyweneKRYVS6xdYALGx4PVcYzmUTWn0CYKYGpKpww1KAaUDF+xypT6gqgq5PKh25NykoT9yGr+hCGzuUstVPz/Hfqx4WgrcYnRBnFywcVmvI4aliu0Qjj4j6i6NBMFw6OsiszYZOutoDAdERiGYDug1Djy5m1cs4dhtCy5JXctQAqxlNJdhkdU5vP9FwrmzsyEqMr4oDIFjvr4GYmtboJJX2fPh5Yigfr7/fpttJDhhbvpfXYxQGOkKJs005vaf5tzta/sxS7sT454x4mBK3ByB2dojM0bw0Pba1hHgqW3BNxow4Y5Vq8AJ14FchkJPwAuTtlKKkGmfCOYLYi2MQIDAQAB"

// out_trade_no:64个字符以内，仅支持字母、数字、下划线
// ./main -addr=x.x.x.x:9090 -out_trade_no=dingdan_2234567890
var (
	addr_ptr     = flag.String("addr", "1.1.1.1:8080", "listen addr")
	out_trade_no = flag.String("out_trade_no", "out_trade_no_unset", "out_trade_no") //订单号每次都要不一样,所以用启动参数设置
)

func main() {
	//aliPayPublicKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1wn1sU/8Q0rYLlZ6sq3enrPZw2ptp6FecHR2bBFLjJ+sKzepROd0bKddgj+Mr1ffr3Ej78mLdWV8IzLfpXUi945DkrQcOUWLY0MHhYVG2jSs/qzFfpzmtut2Cl2TozYpE84zom9ei06u2AXLMBkU6VpznZl+R4qIgnUfByt3Ix5b3h4Cl6gzXMAB1hJrrrCkq+WvWb3Fy0vmk/DUbJEz8i8mQPff2gsHBE1nMPvHVAMw1GMk9ImB4PxucVek4ZbUzVqxZXphaAgUXFK2FSFU+Q+q1SPvHbUsjtIyL+cLA6H/6ybFF9Ffp27Y14AHPw29+243/SpMisbGcj2KD+evBwIDAQAB"
	//appPrivateKey := "MIIEogIBAAKCAQEAy+CRzKw4krA2RzCDTqg5KJg92XkOY0RN3pW4sYInPqnGtHV7YDHu5nMuxY6un+dLfo91OFOEg+RI+WTOPoM4xJtsOaJwQ1lpjycoeLq1OyetGW5Q8wO+iLWJASaMQM/t/aXR/JHaguycJyqlHSlxANvKKs/tOHx9AhW3LqumaCwz71CDF/+70scYuZG/7wxSjmrbRBswxd1Sz9KHdcdjqT8pmieyPqnM24EKBexHDmQ0ySXvLJJy6eu1dJsPIz+ivX6HEfDXmSmJ71AZVqZyCI1MhK813R5E7XCv5NOtskTe3y8uiIhgGpZSdB77DOyPLcmVayzFVLAQ3AOBDmsY6wIDAQABAoIBAHjsNq31zAw9FcR9orQJlPVd7vlJEt6Pybvmg8hNESfanO+16rpwg2kOEkS8zxgqoJ1tSzJgXu23fgzl3Go5fHcoVDWPAhUAOFre9+M7onh2nPXDd6Hbq6v8OEmFapSaf2b9biHnBHq5Chk08v/r74l501w3PVVOiPqulJrK1oVb+0/YmCvVFpGatBcNaefKUEcA+vekWPL7Yl46k6XeUvRfTwomCD6jpYLUhsAKqZiQJhMGoaLglZvkokQMF/4G78K7FbbVLMM1+JDh8zJ/DDVdY2vHREUcCGhl4mCVQtkzIbpxG++vFg7/g/fDI+PquG22hFILTDdtt2g2fV/4wmkCgYEA6goRQYSiM03y8Tt/M4u1Mm7OWYCksqAsU7rzQllHekIN3WjD41Xrjv6uklsX3sTG1syo7Jr9PGE1xQgjDEIyO8h/3lDQyLyycYnyUPGNNMX8ZjmGwcM51DQ/QfIrY/CXjnnW+MVpmNclAva3L33KXCWjw20VsROV1EA8LCL94BUCgYEA3wH4ANpzo7NqXf+2WlPPMuyRrF0QPIRGlFBNtaKFy0mvoclkREPmK7+N4NIGtMf5JNODS5HkFRgmU4YNdupA2I8lIYpD+TsIobZxGUKUkYzRZYZ1m1ttL69YYvCVz9Xosw/VoQ+RrW0scS5yUKqFMIUOV2R/Imi//c5TdKx6VP8CgYAnJ1ADugC4vI2sNdvt7618pnT3HEJxb8J6r4gKzYzbszlGlURQQAuMfKcP7RVtO1ZYkRyhmLxM4aZxNA9I+boVrlFWDAchzg+8VuunBwIslgLHx0/4EoUWLzd1/OGtco6oU1HXhI9J9pRGjqfO1iiIifN/ujwqx7AFNknayG/YkQKBgD6yNgA/ak12rovYzXKdp14Axn+39k2dPp6J6R8MnyLlB3yruwW6NSbNhtzTD1GZ+wCQepQvYvlPPc8zm+t3tl1r+Rtx3ORf5XBZc3iPkGdPOLubTssrrAnA+U9vph61W+OjqwLJ9sHUNK9pSHhHSIS4k6ycM2YAHyIC9NGTgB0PAoGAJjwd1DgMaQldtWnuXjvohPOo8cQudxXYcs6zVRbx6vtjKe2v7e+eK1SSVrR5qFV9AqxDfGwq8THenRa0LC3vNNplqostuehLhkWCKE7Y75vXMR7N6KU1kdoVWgN4BhXSwuRxmHMQfSY7q3HG3rDGz7mzXo1FVMr/uE4iDGm0IXY="
	//初始化支付宝客户端
	//    appId：应用ID
	//    privateKey：应用私钥，支持PKCS1和PKCS8
	//    isProd：是否是正式环境
	flag.Parse()
	addr := *addr_ptr
	var notifyApi = "/alipay/payNotify"
	var notifyUrl = "http://" + addr + notifyApi
	var returnApi = "/alipay/payReturn"
	var returnUrl = "http://" + addr + returnApi

	client, err := alipay.NewClient(appId, appPrivateKey, false)
	if err != nil {
		xlog.Error(err)
		return
	}
	//配置公共参数
	client.SetLocation(alipay.LocationShanghai).
		SetCharset("utf-8").
		SetSignType(alipay.RSA2).
		SetReturnUrl(returnUrl). // 设置网页支付返回后同步调用的URL
		SetNotifyUrl(notifyUrl)  //支付宝平台异步调用的通知URL

	//请求参数
	bm := make(gopay.BodyMap)
	bm.Set("subject", "网站测试支付")
	bm.Set("out_trade_no", *out_trade_no) //订单号每次都要不一样，一样的话，会提示该订单已经成功付款
	bm.Set("total_amount", "88.88")
	bm.Set("product_code", "FAST_INSTANT_TRADE_PAY")

	ctx := context.Background()
	//电脑网站支付请求
	payUrl, err := client.TradePagePay(ctx, bm)
	if err != nil {
		xlog.Error("err:", err)
		return
	}
	xlog.Debug("payUrl:", payUrl)

	// http.HandleFunc(notifyApi, payNotify)
	// http.HandleFunc(returnApi, payNotify)
	// http.ListenAndServe(":8000", nil)

	e := gin.Default()
	e.POST(notifyApi, payNotify)
	e.GET(returnApi, payReturn)
	e.Run(addr)
}

func payNotify(c *gin.Context) {
	//测试是payNotify和payReturn是哪个先,结论是先阿里平台调用notify后，再返回浏览器调用returnUrl
	//这样商家就可以根据支付结果返回自定义的信息给用户。没有returnUrl, 只能支付宝平台展示信息，不够灵活
	time.Sleep(time.Second)
	xlog.Info("sleep over")

	notifyReq, err := alipay.ParseNotifyToBodyMap(c.Request)
	if err != nil {
		xlog.Error("ParseNotifyToBodyMap err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": "payNotify 参数错误"})
		return
	}
	ok, err := alipay.VerifySign(aliPayPublicKey, notifyReq)
	if err != nil {
		xlog.Error("err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": fmt.Sprintf("payNotify VerifySign err:%v", err)})
		return
	}
	msg := ""
	if ok {
		msg = "payNotify 验签成功"
	} else {
		msg = "payNotify 验签失败"
		c.JSON(http.StatusOK, gin.H{"msg": msg})
		return
	}
	fmt.Printf("payNotify msg:%v, notifyReq:%v\n", msg, notifyReq)

	form_trade_status := c.PostForm("trade_status")
	notifyReq_status := notifyReq.Get("trade_status")
	fmt.Printf("form_trade_status:%s, notifyReq_status:%s\n", form_trade_status, notifyReq_status)
	//c.JSON(http.StatusOK, gin.H{"msg": fmt.Sprintf("notifyReq trade_status:%s", notifyReq_status)})
	c.String(http.StatusOK, "%s", "success")
}

func payReturn(c *gin.Context) {
	notifyReq, err := alipay.ParseNotifyToBodyMap(c.Request)
	if err != nil {
		xlog.Error("err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": "payReturn参数错误"})
		return
	}
	ok, err := alipay.VerifySign(aliPayPublicKey, notifyReq)
	if err != nil {
		xlog.Error("err:", err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": fmt.Sprintf("payReturn VerifySign err:%v", err)})
		return
	}
	msg := ""
	if ok {
		msg = "payRetrun 验签成功"
	} else {
		msg = "payRetrun验签失败"
	}
	fmt.Printf("payReturn msg:%v, notifyReq:%v\n", msg, notifyReq)
	c.JSON(http.StatusOK, gin.H{"msg": msg})
}

/*
./main -addr=xxx:9090 -trade_no=dingdan_5234567890
2023/11/12 06:19:20.605307 main.go:73: [DEBUG] >> payUrl: https://openapi-sandbox.dl.alipaydev.com/gateway.do?app_id=9021000122692113&biz_content=%7B%22out_trade_no%22%3A%22dingdan_5234567890%22%2C%22product_code%22%3A%22FAST_INSTANT_TRADE_PAY%22%2C%22subject%22%3A%22%E7%BD%91%E7%AB%99%E6%B5%8B%E8%AF%95%E6%94%AF%E4%BB%98%22%2C%22total_amount%22%3A%2288.88%22%7D&charset=utf-8&format=JSON&method=alipay.trade.page.pay&notify_url=http%3A%2F%2F111.206.113.126%3A9090%2Falipay%2FpayNotify&return_url=http%3A%2F%2F111.206.113.126%3A9090%2Falipay%2FpayReturn&sign=WqJo43K3HLKYnatyT2RbTXY6LH%2BfvV21%2BepvxSbmt%2BTgVHmIGwgdGViTBqE4UXgd2MV47bh0pvNK5o57K2lsECjnDmml9MqXKiTlGuBZk1QGzjKWfPP5m9xuz4OGqDfpidlfPtntd6%2FjH4tb81myhfu%2BHu7e%2BVdRrpUO%2BiqlArboIHpHlffyaCyVM22uJs8jIuSOztq5sgOTijGRYUSdGXy%2BWjnaAuUy%2BipLZ5ia4U5YwQaRwbMAfemrqxl5r5CDiuLIgC6qEQkOveyhtiwuobNU%2FsyFff6ENKoMIJGCOH%2BpDRf%2BuAI%2BQG8XMuxGbRmmIv9d2u%2FuG026gaQvXxshVA%3D%3D&sign_type=RSA2&timestamp=2023-11-12+14%3A19%3A20&version=1.0
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)

[GIN-debug] POST   /alipay/payNotify         --> main.payNotify (3 handlers)
[GIN-debug] GET    /alipay/payReturn         --> main.payReturn (3 handlers)
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on xxx:9090


2023/11/12 06:20:29.801882 main.go:87: [INFO] >> sleep over
payNotify msg:payNotify 验签成功, notifyReq:map[app_id:9021000122692113 auth_app_id:9021000122692113 buyer_id:2088722004255900 buyer_pay_amount:88.88 charset:utf-8 fund_bill_list:[{"amount":"88.88","fundChannel":"ALIPAYACCOUNT"}] gmt_create:2023-11-12 14:20:08 gmt_payment:2023-11-12 14:20:26 invoice_amount:88.88 notify_id:2023111201222142027055900501692676 notify_time:2023-11-12 14:20:28 notify_type:trade_status_sync out_trade_no:dingdan_5234567890 point_amount:0.00 receipt_amount:88.88 seller_id:2088721004255891 subject:网站测试支付 total_amount:88.88 trade_no:2023111222001455900501562422 trade_status:TRADE_SUCCESS version:1.0]
form_trade_status:TRADE_SUCCESS, notifyReq_status:TRADE_SUCCESS
[GIN] 2023/11/12 - 06:20:29 | 200 |  1.001247002s |  119.42.228.161 | POST     "/alipay/payNotify"
payReturn msg:payRetrun 验签成功, notifyReq:map[app_id:9021000122692113 auth_app_id:9021000122692113 charset:utf-8 method:alipay.trade.page.pay.return out_trade_no:dingdan_5234567890 seller_id:2088721004255891 timestamp:2023-11-12 14:20:34 total_amount:88.88 trade_no:2023111222001455900501562422 version:1.0]
[GIN] 2023/11/12 - 06:20:37 | 200 |      584.63µs |   120.229.69.98 | GET      "/alipay/payReturn?charset=utf-8&out_trade_no=dingdan_5234567890&method=alipay.trade.page.pay.return&total_amount=88.88&sign=kniMkwW6QI%2F1rwo%2BzMZXyhiq74uP58Ql%2FXmjokzPak5PSeqeoWrPEKPHaC8ctUTq5VLsapSWOktrQNqaTBur7IrCGetHIu9nxc0MIvw01Vl2eA6bjeDDpkgACiMCifdhUgm%2FI90HssNchEQZZrTKfTxEhaehaqFXPpsOLcjk9iqFztLzLafWAS91UMZiuEQNTNiiGuamFzCNDRqeQEjkDTd%2FnLRt00UGrWMg%2BUKj2qiARV88cbYKZiHtYQQSoTN68w146uLoJBQWTHYFi0ovsOcfN%2Fj%2Bl1piFLZ8I8Gc7TcY6Hj4VQOsQMj1gHDkvEDyct3ArRJvFLkWuSq%2FLGJt4g%3D%3D&trade_no=2023111222001455900501562422&auth_app_id=9021000122692113&version=1.0&app_id=9021000122692113&sign_type=RSA2&seller_id=2088721004255891&timestamp=2023-11-12+14%3A20%3A34"
[GIN] 2023/11/12 - 06:20:38 | 404 |         663ns |   120.229.69.98 | GET      "/favicon.ico"
*/
