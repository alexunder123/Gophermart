package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"

	"gophermart/internal/storage"
)

func (h *Handler) Registration(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Registration read body err")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var newUser username
	if err = json.Unmarshal(bytes, &newUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Debug().Msgf("received new user: %s, %s", newUser.Login, newUser.Password)
	newUser.Password = h.hashPasswd(newUser.Password)
	userID := h.randomID(16)
	log.Debug().Msgf("generated ID, hash: %s, %s", userID, newUser.Password)
	err = h.strg.AddNewUser(newUser.Login, newUser.Password, userID)
	if errors.Is(err, storage.ErrConflict) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("AddNewUser err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// var cookie http.Cookie
	// cookie.Name = "gophermart"
	// cookie.Value = newUser.UserID
	// cookie.Path = "/"
	// cookie.Expires = time.Now().Add(time.Hour)
	// http.SetCookie(w, &cookie)
	w.Header().Add("Authorization", userID)
	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}

func (h *Handler) LogIn(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Login read body err")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var newUser username
	if err = json.Unmarshal(bytes, &newUser); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newUser.Password = h.hashPasswd(newUser.Password)
	userID, err := h.strg.LogInUser(newUser.Login, newUser.Password)
	if errors.Is(err, storage.ErrAuthError) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("LogInUser err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Authorization", userID)
	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}

func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Authorization")
	if userID == "" {
		http.Error(w, "user unauthorized", http.StatusUnauthorized)
		return
	}
	err := h.strg.CheckUser(userID)
	if errors.Is(err, storage.ErrAuthError) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("CheckUser err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Orders read body err")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !h.LynnCheckOrder(bytes) {
		log.Error().Err(err).Msg("lynnCheckOrder err")
		http.Error(w, "lynn Check Order error", http.StatusUnprocessableEntity)
		return
	}

	order := string(bytes)

	err = h.strg.AddNewOrder(userID, order)
	if errors.Is(err, storage.ErrUploaded) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(nil)
		return
	}
	if errors.Is(err, storage.ErrAnotherUserUploaded) {
		log.Error().Err(err).Msg("AddNewOrders err")
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("AddNewOrder err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	w.Write(nil)
}

func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Authorization")
	if userID == "" {
		http.Error(w, "user unauthorized", http.StatusUnauthorized)
		return
	}
	err := h.strg.CheckUser(userID)
	if errors.Is(err, storage.ErrAuthError) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("CheckUser err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Withdraw read body err")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var withdrawEntry userWithdraw
	if err = json.Unmarshal(bytes, &withdrawEntry); err != nil {
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		return
	}
	lynnBz := []byte(withdrawEntry.Order)

	if !h.LynnCheckOrder(lynnBz) {
		log.Error().Err(err).Msg("lynnCheckOrder err")
		http.Error(w, "lynn Check Order error", http.StatusUnprocessableEntity)
		return
	}

	err = h.strg.UserWithdraw(userID, withdrawEntry.Order, withdrawEntry.Sum)
	if errors.Is(err, storage.ErrNotEnouthBalance) {
		log.Error().Err(err).Msg("Withdraw UserWithdraw err")
		http.Error(w, err.Error(), http.StatusPaymentRequired)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("Withdraw UserWithdraw err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}
