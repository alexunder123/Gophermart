package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"gophermart/internal/config"
	"gophermart/internal/storage"
	"math/rand"
	"time"
)

type Handler struct {
	cfg  *config.Config
	strg storage.Storager
}

type username struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type userWithdraw struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func NewHandler(cfg *config.Config, strg storage.Storager) *Handler {
	return &Handler{
		cfg:  cfg,
		strg: strg,
	}
}

func (h *Handler) hashPasswd(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	dst := hash.Sum(nil)
	return hex.EncodeToString(dst)
}

func (h *Handler) randomID(n int) string {
	const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bts := make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < n; i++ {
		bts[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(bts)
}

func (h *Handler) lynnCheckOrder(lynn []int) bool {
	for i := len(lynn) - 2; i >= 0; i -= 2 {
		n := lynn[i] * 2
		if n >= 10 {
			n -= 9
		}
		lynn[i] = n
	}
	sum := 0
	for i := 0; i < len(lynn); i++ {
		sum += lynn[i]
	}
	return sum%10 == 0
}
