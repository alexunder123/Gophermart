package accrualreader

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"gophermart/internal/storage"
)

type AccrualReader struct {
	AccuralSystemAddress string
	ctx                  context.Context
	cancel               context.CancelFunc
	finished             chan struct{}
}

func NewAccrualReader(address string) *AccrualReader {
	ctx, cancel := context.WithCancel(context.Background())
	return &AccrualReader{
		AccuralSystemAddress: address,
		ctx:                  ctx,
		cancel:               cancel,
		finished:             make(chan struct{}),
	}
}

func (ar *AccrualReader) Run(strg storage.Storager) {
	go func() {
		log.Debug().Msg("AccrualReader started")
	loop:
		for {
			select {
			case <-ar.ctx.Done():
				break loop
			default:
				//work
				ordersToUpd, err := strg.GetProcessedOrders()
				if err != nil {
					log.Error().Err(err).Msg("GetProcessedOrders process run error")
					continue
				}
				for _, order := range ordersToUpd {
					log.Debug().Msgf("AccrualReader order: %s", order)
					request, err := http.NewRequest(http.MethodGet, ar.AccuralSystemAddress+"/api/orders/"+order.Order, nil)
					if err != nil {
						log.Error().Err(err).Msg("NewRequest process run error")
						continue
					}
					result, err := http.DefaultClient.Do(request)
					if err != nil {
						log.Error().Err(err).Msg("http.DefaultClient process run error")
						continue
					}
					accuralResultBZ, err := io.ReadAll(result.Body)
					if err != nil {
						log.Error().Err(err).Msg("Read result.Body process run error")
						continue
					}
					defer result.Body.Close()

					log.Debug().Msgf("AccrualReader received status: %d", result.StatusCode)
					if result.StatusCode == 429 {
						t, err := time.ParseDuration(result.Header.Get("Retry-After") + "s")
						if err != nil {
							log.Error().Err(err).Msg("ParseDuration process run error")
							continue
						}
						time.Sleep(t * time.Second)
						continue
					}
					var responce storage.AccuralResult
					if err = json.Unmarshal(accuralResultBZ, &responce); err != nil {
						log.Error().Err(err).Msg("Unmarshal process run error")
						return
					}
					if result.StatusCode == 200 {
						if responce.Status == order.Status {
							continue
						} else {
							responce.UserID = order.UserID
							err := strg.UpdateOrderStatus(responce)
							if err != nil {
								log.Error().Err(err).Msg("GetProcessedOrders UpdateOrderStatus error")
								continue
							}
						}
					}
					if result.StatusCode == 204 {
						err := strg.UpdateOrderStatus(responce)
							if err != nil {
								log.Error().Err(err).Msg("GetProcessedOrders UpdateOrderStatus error")
								continue
							}
						continue
					}
					
				}
			}
			time.Sleep(time.Second)
		}
		close(ar.finished)
		log.Debug().Msg("AccrualReader finished")
	}()
}

func (ar *AccrualReader) Stop() {
	ar.cancel()
	<-ar.finished
}
