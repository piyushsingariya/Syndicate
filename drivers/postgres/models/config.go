package models

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/piyushsingariya/kaku/utils"
)

type Config struct {
	// Hostname of the database.
	//
	// @jsonschema(
	// required=true
	// )
	Host string `json:"host"`
	// Port of the database.
	//
	// @jsonschema(
	// required=true,
	//  minimum=0,
	//  maximum=65536,
	//  default=5432
	// )
	Port int `json:"port"`
	// Name of the database.
	//
	// @jsonschema(
	// required=true
	// )
	Database string `json:"database"`

	// user of the database.
	//
	// @jsonschema(
	// required=true
	// )
	Username string `json:"username"`
	// password of the user.
	//
	// @jsonschema(
	// required=true
	// )
	Password string `json:"password"`
	// JDBC URL Parameters (Advanced)
	//
	// @jsonschema(
	// description="Additional properties to pass to the JDBC URL string when connecting to the database. For more information read about https://jdbc.postgresql.org/documentation/head/connect.html"
	// )
	JDBCURLParams map[string]string `json:"jdbc_url_params"`
	// Hostname of the database.
	//
	// @jsonschema(
	// required=true
	// )
	SSLConfiguration *utils.SSLConfig `json:"ssl"`
}

func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("empty host name")
	} else if strings.Contains(c.Host, "https") || strings.Contains(c.Host, "http") {
		return fmt.Errorf("host should not contain http or https")
	}

	if c.SSLConfiguration == nil {
		return fmt.Errorf("ssl config not set")
	}

	return c.SSLConfiguration.Validate()
}

func (c *Config) ToConnectionString() string {
	// Construct the connection string
	connStr := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s", c.Host, c.Port, c.Database, c.Username, c.Password)

	// Set additional connection parameters if available
	if len(c.JDBCURLParams) > 0 {
		params := ""
		for k, v := range c.JDBCURLParams {
			params += fmt.Sprintf("%s=%s ", pq.QuoteIdentifier(k), pq.QuoteLiteral(v))
		}
		connStr += " options='" + params + "'"
	}

	// Enable SSL if SSLConfig is provided
	if c.SSLConfiguration != nil {
		sslmode := string(c.SSLConfiguration.Mode)
		if sslmode != "" {
			connStr += " sslmode=" + sslmode
		}

		if c.SSLConfiguration.ServerCA != "" {
			connStr += " sslrootcert=" + c.SSLConfiguration.ServerCA
		}

		if c.SSLConfiguration.ClientCert != "" {
			connStr += " sslcert=" + c.SSLConfiguration.ClientCert
		}

		if c.SSLConfiguration.ClientKey != "" {
			connStr += " sslkey=" + c.SSLConfiguration.ClientKey
		}
	}

	return connStr
}
