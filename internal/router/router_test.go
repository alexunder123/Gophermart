package router

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gophermart/internal/accrualreader"
	"gophermart/internal/config"
	"gophermart/internal/handlers"
	"gophermart/internal/logger"
	"gophermart/internal/storage"
)

type username struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func TestRouter(t *testing.T) {
	logger.Newlogger()
	log.Info().Msg("Start test")

	os.Setenv("RUN_ADDRESS", "127.0.0.1:8080")
	os.Setenv("DATABASE_URI", "postgres://postgres:1@localhost:5432/postgres?sslmode=disable")
	os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "postgres://postgres:1@localhost:5432/postgres?sslmode=disable")

	cnfg, err := config.NewConfig()
	require.NoError(t, err)
	strg := storage.NewStorage(cnfg)
	log.Debug().Msg("storage init")
	accrual := accrualreader.NewAccrualReader(cnfg.AccuralSystemAddress)
	accrual.Run(strg)
	hndlr := handlers.NewHandler(cnfg, strg)
	router := NewRouter(hndlr)
	log.Debug().Msg("handler init")

	l, err := net.Listen("tcp", cnfg.RunAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("HTTP error")
	}

	ts := httptest.NewUnstartedServer(router)
	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	// newUser := username{Login: "abpopt88t", Password: "njrto0874NRIY"}
	// newUserBZ, err := json.Marshal(newUser)
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("json.Marshal error")
	// }
	// userID := register(ts, t, newUserBZ)
	// log.Debug().Msgf("received userID: %s", userID)

	now := []byte(`12345678902`)
	bool := hndlr.LynnCheckOrder(now)
	log.Debug().Msgf("result: %t", bool)

	accrual.Stop()
	log.Info().Msg("test finished")
}

func register(ts *httptest.Server, t *testing.T, newUser []byte) string {
	var request *http.Request
	request, err := http.NewRequest(http.MethodPost, ts.URL+"/api/user/register", bytes.NewReader(newUser))
	require.NoError(t, err)
	result, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	assert.Equal(t, 200, result.StatusCode)
	err = result.Body.Close()
	require.NoError(t, err)
	userID := result.Header.Get("gophermart")
	return userID
}
