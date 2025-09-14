package main

import (
	"bytes"
	"dredger/handler"
	"dredger/pkg/conf"
	"dredger/pkg/logger"
	"dredger/service"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"sync" // 导入 sync 包
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var db *gorm.DB

// Upgrader 用于将HTTP连接升级为WebSocket连接
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源的连接，方便开发
		return true
	},
}

// SensorData 定义了发送给前端的数据结构
// 字段名(JSON tag)必须与前端 sensorData 中的 key 一致
type SensorData struct {
	FlowRate               float32 `json:"flowRate"`
	Concentration          float32 `json:"concentration"`
	ProductionRate         float32 `json:"productionRate"`
	LadderDepth            float32 `json:"ladderDepth"`
	CutterRpm              float32 `json:"cutterRpm"`
	SubmergedPumpRpm       float32 `json:"submergedPumpRpm"`
	MudPump1Rpm            float32 `json:"mudPump1Rpm"`
	MudPump2Rpm            float32 `json:"mudPump2Rpm"`
	SubmergedPumpDischarge float32 `json:"submergedPumpDischarge"`
	MudPump1Discharge      float32 `json:"mudPump1Discharge"`
	MudPump2Discharge      float32 `json:"mudPump2Discharge"`
	GpsSpeed               float32 `json:"gpsSpeed"`
	ActualVacuum           float32 `json:"actualVacuum"`
}

// dataIndexMap 将数据字段映射到协议中的AI数据索引 (从0开始)
var dataIndexMap = map[string]int{
	"flowRate": 394, "concentration": 393, "productionRate": 384,
	"ladderDepth": 406, "cutterRpm": 362, "submergedPumpRpm": 339,
	"mudPump1Rpm": 337, "mudPump2Rpm": 338, "submergedPumpDischarge": 365,
	"mudPump1Discharge": 363, "mudPump2Discharge": 364, "gpsSpeed": 372,
	"actualVacuum": 366,
}

// parseSensorData 根据协议解析传感器返回的字节流
func parseSensorData(data []byte) (*SensorData, error) {
	// 协议定义的最小长度：起始符(2)+备用(10)+DI数(2)+AI数(2)+校验(2)+结束符(2)=20
	if len(data) < 20 {
		return nil, fmt.Errorf("data packet too short: %d bytes", len(data))
	}
	// 1. 校验起始符和结束符
	if !bytes.HasPrefix(data, []byte{0x40, 0x01}) {
		return nil, fmt.Errorf("invalid start sequence")
	}
	if !bytes.HasSuffix(data, []byte{0x0D, 0x0A}) {
		return nil, fmt.Errorf("invalid end sequence")
	}

	// 2. 读取DI和AI字节数
	// DI字节数位于索引 12-13
	diByteCount := int(binary.BigEndian.Uint16(data[12:14]))
	// AI字节数位于索引 14-15 (这是AI数据点的数量)
	aiPointCount := int(binary.BigEndian.Uint16(data[14:16]))

	// 3. 校验数据包总长度是否匹配
	// 预期长度 = 头部(16) + DI数据长度 + AI数据长度(点数*4) + 校验和(2) + 结束符(2)
	expectedLength := 16 + diByteCount + (aiPointCount * 4) + 2 + 2
	if len(data) != expectedLength {
		return nil, fmt.Errorf("packet length mismatch: expected %d, got %d", expectedLength, len(data))
	}

	// 4. 解析AI数据
	aiDataStart := 16 + diByteCount
	aiValues := make([]float32, aiPointCount)
	for i := 0; i < aiPointCount; i++ {
		offset := aiDataStart + i*4
		bits := binary.BigEndian.Uint32(data[offset : offset+4])
		aiValues[i] = math.Float32frombits(bits)
	}

	// 5. 填充 SensorData 结构体
	result := &SensorData{}
	// 使用反射或手动映射来填充字段
	if val, ok := dataIndexMap["flowRate"]; ok && val < len(aiValues) {
		result.FlowRate = aiValues[val]
	}
	if val, ok := dataIndexMap["concentration"]; ok && val < len(aiValues) {
		result.Concentration = aiValues[val]
	}
	if val, ok := dataIndexMap["productionRate"]; ok && val < len(aiValues) {
		result.ProductionRate = aiValues[val]
	}
	if val, ok := dataIndexMap["ladderDepth"]; ok && val < len(aiValues) {
		result.LadderDepth = aiValues[val]
	}
	if val, ok := dataIndexMap["cutterRpm"]; ok && val < len(aiValues) {
		result.CutterRpm = aiValues[val]
	}
	if val, ok := dataIndexMap["submergedPumpRpm"]; ok && val < len(aiValues) {
		result.SubmergedPumpRpm = aiValues[val]
	}
	if val, ok := dataIndexMap["mudPump1Rpm"]; ok && val < len(aiValues) {
		result.MudPump1Rpm = aiValues[val]
	}
	if val, ok := dataIndexMap["mudPump2Rpm"]; ok && val < len(aiValues) {
		result.MudPump2Rpm = aiValues[val]
	}
	if val, ok := dataIndexMap["submergedPumpDischarge"]; ok && val < len(aiValues) {
		result.SubmergedPumpDischarge = aiValues[val]
	}
	if val, ok := dataIndexMap["mudPump1Discharge"]; ok && val < len(aiValues) {
		result.MudPump1Discharge = aiValues[val]
	}
	if val, ok := dataIndexMap["mudPump2Discharge"]; ok && val < len(aiValues) {
		result.MudPump2Discharge = aiValues[val]
	}
	if val, ok := dataIndexMap["gpsSpeed"]; ok && val < len(aiValues) {
		result.GpsSpeed = aiValues[val]
	}
	if val, ok := dataIndexMap["actualVacuum"]; ok && val < len(aiValues) {
		result.ActualVacuum = aiValues[val]
	}

	return result, nil
}

// ConnectSensorData WebSocket处理器 (完全重构)
func ConnectSensorData(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()
	log.Println("✅ WebSocket client connected.")

	_, msg, err := ws.ReadMessage()
	if err != nil {
		log.Printf("🔌 WebSocket client disconnected before sending address: %v", err)
		return
	}
	sensorAddr := string(msg)
	log.Printf("Received sensor address from client: %s", sensorAddr)

	tcpConn, err := net.DialTimeout("tcp", sensorAddr, 10*time.Second)
	if err != nil {
		log.Printf("❌ Failed to connect to sensor at %s: %v", sensorAddr, err)
		_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}
	defer tcpConn.Close()
	log.Printf("✅ TCP connection to sensor %s established.", sensorAddr)

	if err := ws.WriteMessage(websocket.TextMessage, []byte("传感器连接成功")); err != nil {
		log.Printf("Failed to send success message to client: %v", err)
		return
	}

	dataChan := make(chan *SensorData)
	done := make(chan struct{})

	// --- Bug修复关键点 ---
	// 使用 sync.Once 来确保关闭 channel 的操作只执行一次
	var once sync.Once
	closeDone := func() {
		close(done)
	}

	// Goroutine 1: 定时发送指令到传感器
	go requestSender(tcpConn, done)

	// Goroutine 2: 从传感器读取并解析数据
	go responseReader(tcpConn, dataChan, closeDone, &once) // 传入 once 和关闭函数

	// Goroutine 3: 监听WebSocket关闭事件
	go wsReader(ws, closeDone, &once) // 传入 once 和关闭函数

	// 主循环：将解析后的数据发送到前端
	for {
		select {
		case data := <-dataChan:
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error marshalling sensor data: %v", err)
				continue
			}
			if err := ws.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				log.Printf("Error writing to WebSocket: %v. Closing connection.", err)
				return
			}
		case <-done:
			log.Println("Connection closing signal received. Shutting down.")
			return
		}
	}
}

// requestSender 定时向TCP连接发送请求指令
func requestSender(conn net.Conn, done chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	command := []byte{0x40, 0xFF, 0x00, 0x00, 0x0D, 0x0A}

	for {
		select {
		case <-ticker.C:
			if _, err := conn.Write(command); err != nil {
				log.Printf("Error writing to TCP sensor: %v", err)
				// 发送失败，可能是连接已断开，关闭goroutine
				return
			}
		case <-done:
			return
		}
	}
}

// responseReader 从TCP连接读取数据，解析后发送到channel
func responseReader(conn net.Conn, dataChan chan<- *SensorData, closeDone func(), once *sync.Once) {
	buffer := make([]byte, 4096) // 4KB 缓冲区
	var packetBuffer []byte
	startMarker := []byte{0x40, 0x01}
	endMarker := []byte{0x0D, 0x0A}

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from TCP sensor: %v", err)
			}
			// --- Bug修复 ---
			// 使用 once.Do 来安全地调用关闭函数
			once.Do(closeDone)
			return
		}

		packetBuffer = append(packetBuffer, buffer[:n]...)

		// 循环处理缓冲区中可能存在的多个完整数据包
		for {
			start := bytes.Index(packetBuffer, startMarker)
			if start == -1 {
				// 没有找到起始符，清空缓冲区等待新数据
				packetBuffer = nil
				break
			}

			// 从起始符开始查找结束符
			end := bytes.Index(packetBuffer[start:], endMarker)
			if end == -1 {
				// 数据包不完整，保留从起始符开始的部分，等待后续数据
				packetBuffer = packetBuffer[start:]
				break
			}

			// 找到了一个完整的数据包
			fullPacketEnd := start + end + len(endMarker)
			packet := packetBuffer[start:fullPacketEnd]

			if parsedData, err := parseSensorData(packet); err == nil {
				dataChan <- parsedData
			} else {
				log.Printf("Failed to parse sensor data packet: %v", err)
			}

			// 移除已处理的数据包，继续处理缓冲区剩余部分
			packetBuffer = packetBuffer[fullPacketEnd:]
		}
	}
}

// wsReader 监听WebSocket的读取，主要用于检测连接是否关闭
func wsReader(ws *websocket.Conn, closeDone func(), once *sync.Once) {
	for {
		// ReadMessage会阻塞，如果客户端断开连接，它会返回一个错误
		if _, _, err := ws.ReadMessage(); err != nil {
			log.Println("🔌 WebSocket client disconnected. Closing TCP connection to sensor.")
			// --- Bug修复 ---
			// 使用 once.Do 来安全地调用关闭函数
			once.Do(closeDone)
			return
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
	config.AllowOrigins = []string{"*"} // 允许所有来源，方便开发
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
		api.GET("/shifts/parameters", h.GetAllShiftParameters)

		// WebSocket路由
		api.GET("/ws/sensor", ConnectSensorData)
	}

	return r
}
