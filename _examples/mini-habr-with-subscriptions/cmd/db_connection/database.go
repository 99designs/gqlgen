package dbconnection

import (
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/rs/zerolog/log"
)

type DatabaseConnection struct {
	dataSourceName string
	DB             *sqlx.DB
}

func NewDatabaseConnection() (*DatabaseConnection, error) {
	log.Info().Msg("Connecting to database")

	log.Debug().Msg("Init database")
	var databaseCon DatabaseConnection

	config, err := newDSN()
	if err != nil {
		return nil, err
	}

	err = databaseCon.connectDatabase(config)
	if err != nil {
		return nil, err
	}

	log.Info().Msg("Сonnection to the database is successful")
	return &databaseCon, err
}

func (db *DatabaseConnection) connectDatabase(config string) error {
	const op = "cmd.dbconnection.connectDatabase()"

	log.Debug().Msg("Attempting to connect to database")

	db.dataSourceName = config

	var connectError error
	db.DB, connectError = sqlx.Connect("pgx", db.dataSourceName)
	if connectError != nil {
		return fmt.Errorf("%s:%w", op, connectError)
	}

	log.Debug().Msg("Connecting to database is successfully")
	return nil
}

func (db *DatabaseConnection) PingDatabase() error {
	const op = "cmd.dbconnection.PingDatabase()"

	log.Info().Msg("Attempting to ping Database")
	if db.DB == nil {
		return fmt.Errorf("%s:%s", op, "database isn`t established")
	}

	var pingError = db.DB.Ping()
	if pingError != nil {
		return fmt.Errorf("%s:%w", op, pingError)
	}

	log.Info().Msg("Ping database is successful")
	return nil
}

func (db *DatabaseConnection) CloseDatabase() error {
	const op = "cmd.dbconnection.CloseDatabase()"

	log.Info().Msg("Attempting to close database")
	var closingError = db.DB.Close()
	if closingError != nil {
		return fmt.Errorf("%s:%w", op, closingError)
	}
	log.Info().Msg("Successful closing of database")
	return nil
}

func newDSN() (string, error) {
	const op = "cmd.dbconnection.NewDSN()"

	log.Debug().Msg("Reading dsn from env variables")
	dsnProtocol := os.Getenv("DB_PROTOCOL")
	if dsnProtocol == "" {
		return "", fmt.Errorf("%s:%s", op, "DB_PROTOCOL isn't set")
	}

	dsnUserName := os.Getenv("DB_USER")
	if dsnUserName == "" {
		return "", fmt.Errorf("%s:%s", op, "DB_USER isn't set")
	}

	dsnPassword := os.Getenv("DB_PASSWORD")
	if dsnPassword == "" {
		return "", fmt.Errorf("%s:%s", op, "DB_PASSWORD isn't set")
	}

	dsnHost := os.Getenv("DB_HOST")
	if dsnHost == "" {
		return "", fmt.Errorf("%s:%s", op, "DB_HOST isn't set")
	}

	dsnPort := os.Getenv("DB_PORT")
	if dsnPort == "" {
		return "", fmt.Errorf("%s:%s", op, "DB_PORT isn't set")
	}

	dsnDBName := os.Getenv("DB_NAME")
	if dsnDBName == "" {
		return "", fmt.Errorf("%s:%s", op, "DB_NAME isn't set")
	}

	dsnOptions := os.Getenv("DB_OPTIONS")
	if dsnOptions == "" {
		return "", fmt.Errorf("%s:%s", op, "DB_OPTIONS isn't set")
	}

	dsn := dsnProtocol + "://" + dsnUserName + ":" + dsnPassword + "@" +
		dsnHost + ":" + dsnPort + "/" + dsnDBName + "?" + dsnOptions

	log.Debug().Msgf("Reading dsn is successful dsn = %s", dsn)
	return dsn, nil
}
