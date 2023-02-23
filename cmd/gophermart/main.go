package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"gophermart/internal/accrualreader"
	"gophermart/internal/config"
	"gophermart/internal/handlers"
	"gophermart/internal/logger"
	"gophermart/internal/router"
	"gophermart/internal/storage"
)

func main() {
	logger.Newlogger()
	log.Info().Msg("Start program")
	cnfg, err := config.NewConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("NewConfig read environment error")
	}
	strg := storage.NewStorage(cnfg)
	log.Debug().Msg("storage init")
	accrual := accrualreader.NewAccrualReader(cnfg.AccuralSystemAddress)
	accrual.Run(strg)
	hndlr := handlers.NewHandler(cnfg, strg)
	router := router.NewRouter(hndlr)
	log.Debug().Msg("handler init")

	go func() {
		err := http.ListenAndServe(cnfg.RunAddress, router)
		if err != nil {
			log.Fatal().Msgf("server failed: %s", err)
		}
	}()

	sigChan := make(chan os.Signal, 10)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for sig := range sigChan {
		switch sig {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			log.Info().Msgf("OS cmd received signal %s", sig)
			accrual.Stop()
			strg.CloseDB()
			os.Exit(0)

		}
	}

}
