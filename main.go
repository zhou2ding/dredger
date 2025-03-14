package main

import (
	"dredger/handler"
	"dredger/pkg/conf"
	"dredger/pkg/logger"
	"dredger/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

var db *gorm.DB

func main() {
	conf.InitConf("./dredger.yaml")
	logger.InitLogger("dredger")

	var err error
	dsn := "root:5023152@tcp(36.133.97.26:26033)/dredger?charset=utf8mb4&parseTime=True&loc=Local"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), gormLogger.Config{
			SlowThreshold: time.Second,
			LogLevel:      gormLogger.Info,
			Colorful:      true,
		}),
	})
	if err != nil {
		logger.Logger.Errorf("failed to connect database: %v", err)
		return
	}

	svc := service.NewService(db)
	r := SetupRouter(svc)
	_ = r.Run(":12580")
}

func SetupRouter(svc *service.Service) *gin.Engine {
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{conf.Conf.GetString("frontend.host")}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	h := handler.NewHandler(svc)
	api := r.Group("/v1")
	{
		api.POST("/data/import", h.ImportData)
		api.GET("/shifts/statistics", h.GetShiftStats)
		api.GET("/data/column/list", h.GetColumns)
		api.GET("/ship/list", h.GetShipList)
		api.GET("/ship/pie", h.GetShipList)
	}

	return r
}
