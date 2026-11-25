// Package auth implements JWT authentication and token revocation in Redis.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

const issuer = "statistical-measures"
const TokenLifetime = 2 * time.Hour
const claimsKey = "jwt_claims"

// Claims identify the user and a unique access token.
type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

type Service struct {
	secret []byte
	redis  *redis.Client
}

func New(ctx context.Context) (*Service, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		log.Println("JWT_SECRET not set: generated a temporary key; tokens expire on server restart")
	}
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes")
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("REDIS_PASSWORD"), DB: 0})
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("Redis unavailable: %w", err)
	}
	return &Service{secret: secret, redis: client}, nil
}

func (s *Service) Close() error { return s.redis.Close() }

func (s *Service) Issue(userID int) (string, error) {
	if userID <= 0 {
		return "", errors.New("invalid user ID")
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	now := time.Now()
	claims := Claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer, Subject: strconv.Itoa(userID), ID: hex.EncodeToString(id),
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(TokenLifetime)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *Service) parse(raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.UserID <= 0 || claims.ID == "" || claims.Subject != strconv.Itoa(claims.UserID) {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func blacklistKey(id string) string { return "blacklist:" + id }

// Revoke stores a token ID in the blacklist until its expiration.
func (s *Service) Revoke(ctx context.Context, claims *Claims) error {
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	return s.redis.Set(ctx, blacklistKey(claims.ID), strconv.Itoa(claims.UserID), ttl).Err()
}

// Middleware allows guests on public endpoints and requires a valid JWT on protected ones.
// Invalid/expired/revoked tokens are treated as guest only on public endpoints.
func (s *Service) Middleware(required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			if required {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "требуется Bearer JWT"})
				return
			}
			c.Next()
			return
		}
		claims, err := s.parse(parts[1])
		if err != nil {
			if required {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "токен недействителен или просрочен"})
				return
			}
			c.Next()
			return
		}
		revoked, err := s.redis.Exists(c.Request.Context(), blacklistKey(claims.ID)).Result()
		if err != nil {
			if required {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "хранилище токенов недоступно"})
				return
			}
			c.Next()
			return
		}
		if revoked != 0 {
			if required {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "токен отозван"})
				return
			}
			c.Next()
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

func CurrentClaims(c *gin.Context) *Claims {
	value, _ := c.Get(claimsKey)
	claims, _ := value.(*Claims)
	return claims
}

func CurrentUserID(c *gin.Context) int {
	if claims := CurrentClaims(c); claims != nil {
		return claims.UserID
	}
	return 0
}
