package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Depado/ginprom"
	"github.com/crm/app/api/routes"
	"github.com/crm/pkg/config"
	"github.com/crm/pkg/database"

	"github.com/crm/pkg/domains/auth"
	"github.com/crm/pkg/domains/whatsapp"
	"github.com/crm/pkg/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func LaunchHttpServer(appc config.App, allows config.Allows) {
	log.Println("Starting HTTP Server...")
	gin.SetMode(gin.DebugMode)

	app := gin.New()
	app.Use(gin.LoggerWithFormatter(func(log gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] - %s \"%s %s %s %d %s\"\n",
			log.TimeStamp.Format("2006-01-02 15:04:05"),
			log.ClientIP,
			log.Method,
			log.Path,
			log.Request.Proto,
			log.StatusCode,
			log.Latency,
		)
	}))
	app.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	app.Use(gin.Recovery())
	app.Use(otelgin.Middleware(appc.Name))
	app.Use(middleware.ClaimIp())
	app.Use(cors.New(cors.Config{
		AllowMethods:     []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With", "Origin", "Accept"},
		AllowOrigins:     []string{"*"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	p := ginprom.New(
		ginprom.Engine(app),
		ginprom.Subsystem("gin"),
		ginprom.Path("/metrics"),
		ginprom.Ignore("/swagger/*any"),
	)
	app.Use(p.Instrument())

	db := database.DBClient()
	api := app.Group("/api/v1")

	// Auth Routes
	auth_repo := auth.NewRepo(db)
	auth_service := auth.NewService(auth_repo)
	routes.AuthRoutes(api.Group("/auth"), auth_service)

	// WhatsApp Routes
	whatsapp_service := whatsapp.NewService()
	routes.WhatsAppRoutes(api.Group("/whatsapp"), whatsapp_service)

	fmt.Println("Server is running on port " + appc.Port)
	if err := app.Run(net.JoinHostPort(appc.Host, appc.Port)); err != nil {
		log.Fatalf("Server başarisiz oldu: %v", err)
	}
}
