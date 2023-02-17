package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"gophermart/internal/handlers"
)

func NewRouter(handler *handlers.Handler) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Post("/api/user/register", handler.Registration)
	router.Post("/api/user/login", handler.LogIn)
	router.Post("/api/user/orders", handler.Orders)
	router.Post("/api/user/balance/withdraw", handler.Withdraw)

	router.Get("/api/user/balance", handler.Balance)
	router.Get("/api/user/orders", handler.OrdersHistory)
	router.Get("/api/user/withdrawals", handler.WithdrawHistory)

	return router
}
