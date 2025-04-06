package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	dbconnection "github.com/nabishec/ozon_habr_api/cmd/db_connection"
	"github.com/nabishec/ozon_habr_api/cmd/server"
	"github.com/nabishec/ozon_habr_api/internal/storage"
	"github.com/nabishec/ozon_habr_api/internal/storage/db"
	inmemory "github.com/nabishec/ozon_habr_api/internal/storage/in-memory"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	debug := flag.Bool("d", false, "set log level to debug")
	easyReading := flag.Bool("r", false, "set console writer")

	var storageType string
	const defaultStorageType = "postgres"
	flag.StringVar(&storageType, "storage", "postgres", "set storage type 'memory'('m') or postgres('p')")
	flag.StringVar(&storageType, "s", "p", "set storage type 'memory'('m') or postgres('p')")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// for reading logs
	if *easyReading {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	switch storageType {
	case "postgres":
	case "memory":
	case "p":
		storageType = "postgres"
	case "m":
		storageType = "memory"
	default:
		log.Error().Msg("Storage type is incorrectly selected")
		storageType = defaultStorageType
		log.Warn().Msg("Default storage type is selected for further work")
	}

	//load enviroments
	err := loadEnv()
	if err != nil {
		log.Error().Err(err).Msg("Don't found configuration")
		os.Exit(1)
	}

	//creating storage according to the settings of the parameters
	storage, err := createStorage(storageType)
	if err != nil {
		log.Error().Err(err).Msg("Failed init storage")
		os.Exit(1)
	}

	server.RunServer(storage)
	//TODO: RUN SERVER
}

func loadEnv() error {
	const op = "cmd.loadEnv()"
	err := godotenv.Load(".env")
	if err != nil {
		return fmt.Errorf("%s:%s", op, "failed load env file")
	}
	return nil
}

func createStorage(storageType string) (storage.StorageImp, error) {

	if storageType == "memory" {
		return createResolverInMemory()

	} else {
		return createResolverWithDB()
	}
}

func createResolverInMemory() (storage.StorageImp, error) {
	const op = "cmd.createResolverInMemory()"

	inmemory := inmemory.NewStorage()

	return inmemory, nil
}

func createResolverWithDB() (storage.StorageImp, error) {
	const op = "cmd.createDBStorage()"
	dbConn, err := dbconnection.NewDatabaseConnection()
	if err != nil {
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	cacheConn, err := dbconnection.NewCacheConnection()
	if err != nil {
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	storage := db.NewStorage(dbConn.DB, cacheConn.Cache)

	return storage, nil
}
