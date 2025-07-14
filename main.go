package main

import (
	"dredger/handler"
	"dredger/pkg/conf"
	"dredger/pkg/logger"
	"dredger/service"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

var db *gorm.DB

// Upgrader 用于将HTTP连接升级为WebSocket连接
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源的连接，方便开发
		return true
	},
}

// ConnectSensorData WebSocket处理器 (连接版)
// 负责与前端建立WS连接，并根据前端指令与传感器建立TCP长连接
func ConnectSensorData(c *gin.Context) {
	// 1. 升级HTTP连接为WebSocket连接
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()
	log.Println("✅ WebSocket client connected.")

	// 2. 等待并读取前端发送的传感器地址
	_, msg, err := ws.ReadMessage()
	if err != nil {
		log.Printf("🔌 WebSocket client disconnected before sending address: %v", err)
		return
	}
	sensorAddr := string(msg)
	log.Printf("Received sensor address from client: %s", sensorAddr)

	// 3. 根据地址连接到TCP传感器
	tcpConn, err := net.DialTimeout("tcp", sensorAddr, 10*time.Second) // 增加超时时间
	if err != nil {
		log.Printf("❌ Failed to connect to sensor at %s: %v", sensorAddr, err)
		// 向前端发送错误信息
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return // 错误，关闭WS连接
	}
	// 当函数退出时（即WS断开时），确保关闭TCP连接
	defer tcpConn.Close()
	log.Printf("✅ TCP connection to sensor %s established.", sensorAddr)

	// 4. 向前端发送成功消息
	successMsg := "传感器连接成功"
	if err := ws.WriteMessage(websocket.TextMessage, []byte(successMsg)); err != nil {
		log.Printf("Failed to send success message to client: %v", err)
		return
	}

	// 5. 进入循环，保持连接，不进行任何数据交互
	// 仅用于侦听前端的断开事件
	log.Println("Maintaining long connections. Waiting for client to disconnect...")
	for {
		// ReadMessage会阻塞，直到有消息或连接断开
		if _, _, err := ws.ReadMessage(); err != nil {
			log.Println("🔌 WebSocket client disconnected. Closing TCP connection to sensor.")
			break // 退出循环，触发defer的关闭操作
		}
	}
}

func main() {
	if _, err := os.Stat("tmp"); os.IsNotExist(err) {
		if err = os.Mkdir("tmp", 0755); err != nil {
			log.Fatalf("Failed to create tmp directory: %v", err)
		}
	}

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
		api.GET("data-Tirange/nonempty", h.GetNoneEmptyTimeRange)
		api.POST("/data/theory/optimal", h.SetTheoryOptimal)
		api.GET("/data/theory/optimal", h.GetTheoryOptimal)
		api.GET("/shifts/parameters", h.GetAllShiftParameters)

		// WebSocket路由
		api.GET("/ws/sensor", ConnectSensorData)
	}

	return r
}
