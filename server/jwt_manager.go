package main

import (
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/mjafari98/go-auth/models"
	"github.com/mjafari98/go-auth/pb"
	"os"
	"strings"
	"time"
)

type JWTManager struct {
	tokenDuration time.Duration
}

type UserClaims struct {
	jwt.StandardClaims
	UserID    uint64 `json:"user_id"`
	Username  string `json:"username"`
	RoleID    uint64 `json:"role"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

func (manager *JWTManager) Generate(user *models.User) *pb.JWTToken {
	claims := UserClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(manager.tokenDuration).Unix(),
		},
		UserID:    user.ID,
		Username:  user.Username,
		RoleID:    user.RoleID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES512, claims)

	key, _ := base64.StdEncoding.DecodeString(os.Getenv("PRIVATE_KEY"))
	privateKey, err := jwt.ParseECPrivateKeyFromPEM(key)
	if err != nil {
		panic(err)
	}

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}

	return &pb.JWTToken{Token: signedToken}
}

func (manager *JWTManager) Verify(jwtToken string) (*UserClaims, error) {
	var err error

	key, _ := base64.StdEncoding.DecodeString(os.Getenv("PUBLIC_KEY"))
	var publicKey *ecdsa.PublicKey
	if publicKey, err = jwt.ParseECPublicKeyFromPEM(key); err != nil {
		return nil, fmt.Errorf("unable to parse ECDSA public key: %v", err)
	}

	parts := strings.Split(jwtToken, ".")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid token")
	}

	err = jwt.SigningMethodES512.Verify(strings.Join(parts[0:2], "."), parts[2], publicKey)
	if err != nil {
		return nil, fmt.Errorf("error while verifying key: %v", err)
	}

	token, err := jwt.ParseWithClaims(jwtToken, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("invalid claims: %v", ok)
	}
}

func NewJWTManager(tokenDuration time.Duration) *JWTManager {
	return &JWTManager{tokenDuration}
}
