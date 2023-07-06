package main

import (
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/piyushsingariya/kaku"
	"github.com/piyushsingariya/kaku/drivers/postgres/driver"
	"github.com/piyushsingariya/kaku/logger"
	"github.com/piyushsingariya/kaku/safego"
)

func main() {
	defer safego.Recovery()

	driver := &driver.Postgres{}
	defer driver.CloseConnection()

	cmd, err := kaku.RegisterDriver(driver)
	if err != nil {
		logger.Fatal(err)
	}

	// Execute the root command
	err = cmd.Execute()
	if err != nil {
		logger.Fatal(err)
	}
}
