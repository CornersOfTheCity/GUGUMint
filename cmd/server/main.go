package main

import (
	"log"
	"os"

	"GUGUMint/internal/config"
	"GUGUMint/internal/db"
	"GUGUMint/internal/eth"
	"GUGUMint/internal/httpserver"
	"GUGUMint/internal/service"
)

func main() {
	cfg := config.Load()

	dbConn, err := db.NewPostgres(cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	ethClient, chainID, err := eth.NewClient(cfg.BSCRPCURL, cfg.ChainID)
	if err != nil {
		log.Fatalf("failed to create eth client: %v", err)
	}

	contract, err := eth.NewMintContract(ethClient, cfg.ContractAddress, chainID, cfg.PrivateKey)
	if err != nil {
		log.Fatalf("failed to init contract: %v", err)
	}

	mintService := service.NewMintService(dbConn, contract)

	r := httpserver.NewRouter(mintService)

	addr := ":8080"
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		addr = v
	}

	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
