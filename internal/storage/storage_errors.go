package storage

import "errors"

var (
	ErrNoContent           error = errors.New("StatusNoContent")
	ErrConflict            error = errors.New("StatusConflict")
	ErrAuthError           error = errors.New("StatusAuthorizationError")
	ErrGone                error = errors.New("StatusGone")
	ErrUploaded            error = errors.New("OrdersUpladedEarlier")
	ErrAnotherUserUploaded error = errors.New("OrdersUpladedByAnotherUser")
	ErrNotEnouthBalance           error = errors.New("OrdersPaymentRequired")
)
