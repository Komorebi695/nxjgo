package token

import (
	"errors"
	"github.com/Komorebi695/nxjgo"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"time"
)

const JWTToken = "jwt_token"

type JwtHandler struct {
	// 算法
	Alg string
	// 登录认证
	Authenticator func(ctx *nxjgo.Context) (map[string]any, error)
	// 过期时间 秒
	TimeOut        time.Duration
	RefreshTimeOut time.Duration
	// 时间函数 从此时开始计算过期
	TimeFuc func() time.Time
	//私钥
	PrivateKey string
	//key
	Key            []byte
	RefreshKey     string
	SendCookie     bool
	CookieName     string
	CookieMaxAge   int
	CookieDomain   string
	SecureCookie   bool
	CookieHTTPOnly bool
	Handler        string
	AuthHandler    func(ctx *nxjgo.Context, err error)
}

type JWTResponse struct {
	Token        string
	RefreshToken string
}

func (j *JwtHandler) LoginHandler(ctx *nxjgo.Context) (*JWTResponse, error) {
	data, err := j.Authenticator(ctx)
	if err != nil {
		return nil, err
	}
	if j.Alg == "" {
		j.Alg = "HS256"
	}
	// A
	signingMethod := jwt.GetSigningMethod(j.Alg)
	token := jwt.New(signingMethod)
	// B
	claims := token.Claims.(jwt.MapClaims)
	if data != nil {
		for k, v := range data {
			claims[k] = v
		}
	}
	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFuc().Add(j.TimeOut)
	claims["exp"] = expire.Unix()
	claims["iat"] = j.TimeFuc().Unix()
	var tokenString string
	var tokenError error
	// C
	if j.usingPublicKeyAlgo() {
		tokenString, tokenError = token.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenError = token.SignedString(j.Key)
	}
	if tokenError != nil {
		return nil, tokenError
	}
	refreshToken, err := j.refreshToken(token)
	if err != nil {
		return nil, err
	}
	jr := &JWTResponse{}
	jr.Token = tokenString
	jr.RefreshToken = refreshToken
	// send cookie
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = int(expire.Unix() - j.TimeFuc().Unix())
		}
		maxAge := j.CookieMaxAge
		ctx.SetCookie(j.CookieName, tokenString, maxAge, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	return jr, nil
}

func (j *JwtHandler) usingPublicKeyAlgo() bool {
	switch j.Alg {
	case "RS256", "RS512", "RS384":
		return true
	}
	return false
}

func (j *JwtHandler) LogoutHandler(ctx *nxjgo.Context) error {
	// 清除cookie即可
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		ctx.SetCookie(j.CookieName, "", -1, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
		return nil
	}
	return nil
}

func (j *JwtHandler) refreshToken(token *jwt.Token) (string, error) {
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = j.TimeFuc().Add(j.RefreshTimeOut).Unix()
	var tokenString string
	var tokenError error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenError = token.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenError = token.SignedString(j.Key)
	}
	if tokenError != nil {
		return "", tokenError
	}
	return tokenString, nil
}

func (j *JwtHandler) RefreshHandler(ctx *nxjgo.Context) (*JWTResponse, error) {
	var token string
	// 检测refresh token是否过期
	storageToken, exists := ctx.Get(j.RefreshKey)
	if exists {
		token = storageToken.(string)
	}
	if token == "" {
		return nil, errors.New("token not exist")
	}
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if j.usingPublicKeyAlgo() {
			return []byte(j.PrivateKey), nil
		}
		return []byte(j.Key), nil
	})
	if err != nil {
		return nil, err
	}
	claims := t.Claims.(jwt.MapClaims)
	// 未过期的情况下，重新生成token 和 refreshToken
	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFuc().Add(j.RefreshTimeOut)
	claims["exp"] = expire.Unix()
	claims["iat"] = j.TimeFuc().Unix()
	var tokenString string
	var tokenError error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenError = t.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenError = t.SignedString(j.Key)
	}
	if tokenError != nil {
		return nil, tokenError
	}
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = int(expire.Unix() - j.TimeFuc().Unix())
		}
		maxAge := j.CookieMaxAge
		ctx.SetCookie(j.CookieName, tokenString, maxAge, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	refreshToken, err := j.refreshToken(t)
	if err != nil {
		return nil, err
	}
	jr := &JWTResponse{}
	jr.Token = tokenString
	jr.RefreshToken = refreshToken
	return jr, nil
}

func (j *JwtHandler) AuthInterceptor(next nxjgo.HandlerFunc) nxjgo.HandlerFunc {
	return func(ctx *nxjgo.Context) {
		if j.Handler == "" {
			j.Handler = "Authorization"
		}
		token := ctx.R.Header.Get(j.Handler)
		if token == "" && j.SendCookie {
			cookie, err := ctx.GetCookie(j.CookieName)
			if err != nil {
				j.AuthErrorHandler(ctx, err)
				return
			}
			token = cookie
		} else if token == "" {
			j.AuthErrorHandler(ctx, errors.New("token is null"))
			return
		}
		t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			if j.usingPublicKeyAlgo() {
				return []byte(j.PrivateKey), nil
			}
			return []byte(j.Key), nil
		})
		if err != nil {
			j.AuthErrorHandler(ctx, err)
			return
		}
		ctx.Set("jwt_claims", t.Claims.(jwt.MapClaims))
		next(ctx)
	}
}

func (j *JwtHandler) AuthErrorHandler(ctx *nxjgo.Context, err error) {
	if j.Authenticator == nil {
		ctx.W.WriteHeader(http.StatusUnauthorized)
	} else {
		j.AuthHandler(ctx, err)
	}
}
