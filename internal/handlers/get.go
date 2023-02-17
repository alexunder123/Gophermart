package handlers

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"gophermart/internal/storage"
)

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {
	userID := w.Header().Get("gophermart")
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
		log.Error().Err(err).Msg("OrdersHistory CheckUser err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// var balance userBalance
	balance, err := h.strg.UserBalance(userID)
	if err != nil {
		log.Error().Err(err).Msg("Balance UserBalance err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(balance)
}

func (h *Handler) OrdersHistory(w http.ResponseWriter, r *http.Request) {
	userID := w.Header().Get("gophermart")
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
		log.Error().Err(err).Msg("OrdersHistory CheckUser err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orders, err := h.strg.UserOrders(userID)
	if errors.Is(err, storage.ErrNoContent) {
		http.Error(w, err.Error(), http.StatusNoContent)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("OrdersHistory UserOrders err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(orders)
}

func (h *Handler) WithdrawHistory(w http.ResponseWriter, r *http.Request) {
	userID := w.Header().Get("gophermart")
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
		log.Error().Err(err).Msg("WithdrawHistory CheckUser err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	withdraws, err := h.strg.UserWithdrawals(userID)
	if errors.Is(err, storage.ErrNoContent) {
		http.Error(w, err.Error(), http.StatusNoContent)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("WithdrawHistory UserWithdrawals err")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(withdraws)
}