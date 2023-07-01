package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// 定义JWT密钥
var jwtKey = []byte("secret_key")

// 定义用户结构体
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// 用户登录处理函数
func loginHandler(w http.ResponseWriter, r *http.Request) {
	// 解析用户名和密码
	username := r.FormValue("username")
	password := r.FormValue("password")

	// 在实际应用中，通常需要从数据库或其他存储中获取用户信息进行验证

	// 假设用户名和密码验证成功，生成JWT令牌
	user := User{
		ID:       1,
		Username: username,
		Password: password,
	}

	token, err := generateToken(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println("token:", token)
	// 将令牌返回给用户
	w.Write([]byte(token))
}

// 生成JWT令牌
func generateToken(user User) (string, error) {
	// 定义令牌的过期时间
	expirationTime := time.Now().Add(15 * time.Minute)

	// 创建令牌声明
	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		IssuedAt:  time.Now().Unix(),
		Subject:   fmt.Sprintf("%d", user.ID),
	}

	// 创建令牌对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// 保护的受限资源处理函数
func restrictedHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求头中获取令牌
	tokenString := r.Header.Get("Authorization")

	// 解析令牌
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证令牌签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return jwtKey, nil
	})

	// 验证令牌有效性
	if err != nil || !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 从令牌中提取用户ID
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID := claims["sub"].(string)

	// 在这里可以进行进一步的权限验证和业务处理

	// 返回受限资源
	w.Write([]byte(fmt.Sprintf("Restricted resource accessed by user ID: %s", userID)))
}

func main() {
	// 定义路由和处理函数
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/restricted", restrictedHandler)

	// 启动HTTP服务器
	log.Fatal(http.ListenAndServe(":8080", nil))
}
