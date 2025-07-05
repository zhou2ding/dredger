package main

import (
	"dredger/handler"
	"dredger/pkg/conf"
	"dredger/pkg/logger"
	"dredger/service"
	"fmt"
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

	host := conf.Conf.GetString("database.host")
	password := conf.Conf.GetString("database.password")
	var err error
	dsn := fmt.Sprintf("root:%s@tcp(%s)/dredger?charset=utf8mb4&parseTime=True&loc=Local", password, host)
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
		api.GET("/data/column/list/:shipName", h.GetColumns)
		api.GET("/ship/list", h.GetShipList)
		api.GET("/shifts/optimal", h.GetOptimalShift)
		api.GET("/data/replay/:columnName", h.GetHistoryData)
		api.GET("data/timerange/global", h.GetGlobalTimeRange)
		api.GET("data/timerange/nonempty", h.GetNoneEmptyTimeRange)
		api.POST("/data/theory/optimal", h.SetTheoryOptimal)
		api.GET("/data/theory/optimal", h.GetTheoryOptimal)
	}

	return r
}
