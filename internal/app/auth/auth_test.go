package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestIssueAndParse(t *testing.T) {
	s := &Service{secret: []byte("test-secret-only-not-for-production")}
	raw, err := s.Issue(7)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.parse(raw)
	if err != nil || c.UserID != 7 || c.ID == "" {
		t.Fatalf("invalid claims: %v", err)
	}
	other := &Service{secret: []byte("different-secret-not-for-production")}
	if _, err := other.parse(raw); err == nil {
		t.Fatal("accepted wrong signature")
	}
}

func TestExpiredToken(t *testing.T) {
	s := &Service{secret: []byte("test-secret-only-not-for-production")}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: "1", ID: "test", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}).SignedString(s.secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.parse(raw); err == nil {
		t.Fatal("accepted expired token")
	}
}

func TestGuestRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Service{}
	r := gin.New()
	r.GET("/public", s.Middleware(false), func(c *gin.Context) {
		if CurrentUserID(c) != 0 {
			t.Error("guest is not anonymous")
		}
		c.Status(http.StatusOK)
	})
	r.PUT("/private", s.Middleware(true), func(c *gin.Context) { c.Status(http.StatusOK) })
	for _, tc := range []struct {
		method, path, header string
		status               int
	}{
		{"GET", "/public", "", 200}, {"GET", "/public", "Bearer invalid", 200}, {"PUT", "/private", "", 401}, {"PUT", "/private", "Bearer invalid", 401},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", tc.header)
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Errorf("%s %s: got %d want %d", tc.method, tc.path, w.Code, tc.status)
		}
	}
}
