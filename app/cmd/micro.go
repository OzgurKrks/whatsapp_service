package cmd

import (
	"github.com/crm/pkg/config"
	"github.com/crm/pkg/database"
	"github.com/crm/pkg/server"
	"github.com/crm/pkg/utils"
)

func StartApp() {
	config := config.InitConfig()
	utils.LoadEnv()
	database.InitDB(config.Database)
	server.LaunchHttpServer(config.App, config.Allows)
}
