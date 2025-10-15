package main

import (
	"bytes"
	"dredger/handler"
	"dredger/model" // 导入 model 包
	"dredger/pkg/conf"
	"dredger/pkg/logger"
	"dredger/service"
	"encoding/binary"
	"fmt"
	"io"
	"log"
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

// SensorData 定义了发送给前端的完整数据结构
// 该结构严格参照 dredger_data_hl.gen.go 文件生成，确保字段和json标签完全一致
type SensorData struct {
	// PredictedVacuum 是唯一计算得出的字段
	PredictedVacuum float64 `json:"predictedVacuum"` // (计算值) 预估真空度 (kPa)

	// --- 以下所有字段均严格来自 model.DredgerDataHl 并与协议对应 ---
	LeftEarDraft                        float64 `json:"left_ear_draft"`                          // 327: 左耳轴吃水(m)
	UnderwaterPumpSuctionSealPressure   float64 `json:"underwater_pump_suction_seal_pressure"`   // 328: 水下泵吸入端封水压力(bar)
	UnderwaterPumpShaftSealPressure     float64 `json:"underwater_pump_shaft_seal_pressure"`     // 329: 水下泵轴端封水压力(bar)
	MudPump1ShaftSealPressure           float64 `json:"mud_pump_1_shaft_seal_pressure"`          // 330: 1#泥泵轴端封水压力(bar)
	MudPump1SuctionSealPressure         float64 `json:"mud_pump_1_suction_seal_pressure"`        // 331: 1#泥泵吸入端封水压力(bar)
	MudPump2SuctionSealPressure         float64 `json:"mud_pump_2_suction_seal_pressure"`        // 332: 2#泥泵吸入端封水压力(bar)
	MudPump2ShaftSealPressure           float64 `json:"mud_pump_2_shaft_seal_pressure"`          // 333: 2#泥泵轴端封水压力(bar)
	RightEarDraft                       float64 `json:"right_ear_draft"`                         // 334: 右耳轴吃水(m)
	LeftAnchorRodAngle                  float64 `json:"left_anchor_rod_angle"`                   // 335: 左抛锚杆角度传感器(°)
	RightAnchorRodAngle                 float64 `json:"right_anchor_rod_angle"`                  // 336: 右抛锚杆角度传感器(°)
	MudPump1Speed                       float64 `json:"mud_pump_1_speed"`                        // 337: 1#泥泵转速(rpm)
	MudPump2Speed                       float64 `json:"mud_pump_2_speed"`                        // 338: 2#泥泵转速(rpm)
	UnderwaterPumpSpeed                 float64 `json:"underwater_pump_speed"`                   // 339: 水下泵转速(rpm)
	FlowVelocity                        float64 `json:"flow_velocity"`                           // 340: 流速(m/s)
	Density                             float64 `json:"density"`                                 // 341: 密度(t/m3)
	UnderwaterPumpMotorCurrent          float64 `json:"underwater_pump_motor_current"`           // 342: 水下泵电机电流(A)
	UnderwaterPumpMotorVoltage          float64 `json:"underwater_pump_motor_voltage"`           // 343: 水下泵电机电压(V)
	UnderwaterPumpTorque                float64 `json:"underwater_pump_torque"`                  // 344: 水下泵扭矩(KN)
	UnderwaterPumpMotorSpeed            float64 `json:"underwater_pump_motor_speed"`             // 345: 水下泵电机转速(rpm)
	MudPump2DieselLoad                  float64 `json:"mud_pump_2_diesel_load"`                  // 346: 2#泥泵柴油机负荷(mm)
	MudPump2DieselSpeed                 float64 `json:"mud_pump_2_diesel_speed"`                 // 347: 2#泥泵柴油机转速(rpm)
	MudPump1DieselLoad                  float64 `json:"mud_pump_1_diesel_load"`                  // 348: 1#泥泵柴油机负荷(mm)
	MudPump1DieselSpeed                 float64 `json:"mud_pump_1_diesel_speed"`                 // 349: 1#泥泵柴油机转速(rpm)
	HydraulicPumpDieselLoad             float64 `json:"hydraulic_pump_diesel_load"`              // 350: 液压泵柴油机负荷(mm)
	HydraulicPumpDieselSpeed            float64 `json:"hydraulic_pump_diesel_speed"`             // 351: 液压泵柴油机转速(rpm)
	GateValveFlushPressure              float64 `json:"gate_valve_flush_pressure"`               // 352: 闸阀冲洗压力(bar)
	CutterBearingFlushPressure          float64 `json:"cutter_bearing_flush_pressure"`           // 353: 绞刀轴承冲水压力(bar)
	TrolleyHydraulicCylinderPressure    float64 `json:"trolley_hydraulic_cylinder_pressure"`     // 354: 台车液压油缸压力(bar)
	SteelPileHydraulicCylinderPressure  float64 `json:"steel_pile_hydraulic_cylinder_pressure"`  // 355: 钢桩液压油缸压力(bar)
	GateValveSystemPressure             float64 `json:"gate_valve_system_pressure"`              // 356: 闸阀系统压力(bar)
	RightTransversePressure             float64 `json:"right_transverse_pressure"`               // 357: 压力传感器（右横移压力）(bar)
	LeftTransversePressure              float64 `json:"left_transverse_pressure"`                // 358: 压力传感器（左横移压力）(bar)
	TrolleyTravel                       float64 `json:"trolley_travel"`                          // 359: 台车行程(m)
	LeftTransverseSpeed                 float64 `json:"left_transverse_speed"`                   // 360: 左横移速度(m/min)
	RightTransverseSpeed                float64 `json:"right_transverse_speed"`                  // 361: 右横移速度(m/min)
	CutterSpeed                         float64 `json:"cutter_speed"`                            // 362: 绞刀转速(rpm)
	MudPump1DischargePressure           float64 `json:"mud_pump_1_discharge_pressure"`           // 363: 1#泥泵排出压力(bar)
	MudPump2DischargePressure           float64 `json:"mud_pump_2_discharge_pressure"`           // 364: 2#泥泵排出压力(bar)
	UnderwaterPumpDischargePressure     float64 `json:"underwater_pump_discharge_pressure"`      // 365: 水下泵排出压力(bar)
	UnderwaterPumpSuctionVacuum         float64 `json:"underwater_pump_suction_vacuum"`          // 366: 水下泵吸入真空(bar)
	BridgeAngle                         float64 `json:"bridge_angle"`                            // 367: 桥架角度(°)
	CompassAngle                        float64 `json:"compass_angle"`                           // 368: 罗经角度(°)
	Gps1X                               float64 `json:"gps1_x"`                                  // 369: GPS1_X
	Gps1Y                               float64 `json:"gps1_y"`                                  // 370: GPS1_Y
	Gps1Heading                         float64 `json:"gps1_heading"`                            // 371: GPS1航向(°)
	Gps1Speed                           float64 `json:"gps1_speed"`                              // 372: GPS1航速(m/s)
	TideLevel                           float64 `json:"tide_level"`                              // 373: 潮位(m)
	WaterDensity                        float64 `json:"water_density"`                           // 374: 水密度(t/m3)
	FieldSlurryDensity                  float64 `json:"field_slurry_density"`                    // 375: 现场泥浆比重
	TrimAngle                           float64 `json:"trim_angle"`                              // 376: 横倾角度(°)
	PitchAngle                          float64 `json:"pitch_angle"`                             // 377: 纵倾角度(°)
	CompassRadian                       float64 `json:"compass_radian"`                          // 378: 罗经弧度(rad)
	Gps1Latitude                        float64 `json:"gps1_latitude"`                           // 379: GPS1_纬度
	Gps1Longitude                       float64 `json:"gps1_longitude"`                          // 380: GPS1_经度
	EarDraft                            float64 `json:"ear_draft"`                               // 381: 耳轴吃水(m)
	TransverseSpeed                     float64 `json:"transverse_speed"`                        // 382: 横移速度(m/min)
	HourlyOutputRate                    float64 `json:"hourly_output_rate"`                      // 384: 小时产量率
	RotationRadius                      float64 `json:"rotation_radius"`                         // 385: 旋转半径(m)
	CutterX                             float64 `json:"cutter_x"`                                // 386: 绞刀x
	CutterY                             float64 `json:"cutter_y"`                                // 387: 绞刀y
	CurrentShiftOutput                  float64 `json:"current_shift_output"`                    // 388: 上一班组产量 -> Mapped to CurrentShiftOutput as PreviousShiftProduction is not in model
	CurrentShiftOutputRate              float64 `json:"current_shift_output_rate"`               // 389: 当前班产量
	OutletFlowVelocity                  float64 `json:"outlet_flow_velocity"`                    // 390: 出口流速
	LeftTransverseTorque                float64 `json:"left_transverse_torque"`                  // 391: 左横移扭矩(KN)
	CutterTorque                        float64 `json:"cutter_torque"`                           // 392: 绞刀扭矩(KN)
	Concentration                       float64 `json:"concentration"`                           // 393: 浓度(t/m3)
	FlowRate                            float64 `json:"flow_rate"`                               // 394: 流量(m3/h)
	RightTransverseTorque               float64 `json:"right_transverse_torque"`                 // 395: 右横移扭矩(KN)
	LeftAnchorWinchSpeed                float64 `json:"left_anchor_winch_speed"`                 // 396: 左起锚绞车速度
	LeftAnchorWinchTorque               float64 `json:"left_anchor_winch_torque"`                // 397: 左起锚绞车扭矩
	RightAnchorWinchSpeed               float64 `json:"right_anchor_winch_speed"`                // 398: 右起锚绞车速度
	RightAnchorWinchTorque              float64 `json:"right_anchor_winch_torque"`               // 399: 右起锚绞车扭矩
	LeftSwingWinchSpeed                 float64 `json:"left_swing_winch_speed"`                  // 400: 左回转绞车速度
	LeftSwingWinchTorque                float64 `json:"left_swing_winch_torque"`                 // 401: 左回转绞车扭矩
	RightSwingWinchSpeed                float64 `json:"right_swing_winch_speed"`                 // 402: 右回转绞车速度
	RightSwingWinchTorque               float64 `json:"right_swing_winch_torque"`                // 403: 右回转绞车扭矩
	BridgeWinchSpeed                    float64 `json:"bridge_winch_speed"`                      // 404: 起桥绞车速度
	BridgeWinchTorque                   float64 `json:"bridge_winch_torque"`                     // 405: 起桥绞车扭矩
	BridgeDepth                         float64 `json:"bridge_depth"`                            // 406: 桥架深度(m)
	TransverseDirection                 int32   `json:"transverse_direction"`                    // 407: 横移方向
	CutterCuttingAngle                  float64 `json:"cutter_cutting_angle"`                    // 408: 绞刀切削角
	UnderwaterPumpPower                 float64 `json:"underwater_pump_power"`                   // 409: 水下泵功率
	MudPump1Power                       float64 `json:"mud_pump_1_power"`                        // 410: 1#泥泵功率
	MudPump2Power                       float64 `json:"mud_pump_2_power"`                        // 411: 2#泥泵功率
	UnderwaterPumpShaftPower            float64 `json:"underwater_pump_shaft_power"`             // 412: 水下泵轴端驱动功率
	MudPump1ShaftPower                  float64 `json:"mud_pump_1_shaft_power"`                  // 413: 1#泥泵轴端驱动功率
	MudPump2ShaftPower                  float64 `json:"mud_pump_2_shaft_power"`                  // 414: 2#泥泵轴端驱动功率
	UnderwaterPumpEfficiency            float64 `json:"underwater_pump_efficiency"`              // 415: 水下泵泵效
	MudPump1Efficiency                  float64 `json:"mud_pump_1_efficiency"`                   // 416: 1#泥泵泵效
	MudPump2Efficiency                  float64 `json:"mud_pump_2_efficiency"`                   // 417: 2#泥泵泵效
	PipelineAverageConcentration        float64 `json:"pipeline_average_concentration"`          // 418: 管路平均浓度
	PipelineTotalDamping                float64 `json:"pipeline_total_damping"`                  // 419: 管路总阻尼
	DensityForecast                     float64 `json:"density_forecast"`                        // 420: 密度预报值
	CuttingThickness                    float64 `json:"cutting_thickness"`                       // 421: 切削厚度
	ShipDirection                       float64 `json:"ship_direction"`                          // 422: 船体方向
	Gps1SignalQuality                   float64 `json:"gps1_signal_quality"`                     // 423: 1#GPS信号质量
	Gps2SignalQuality                   float64 `json:"gps2_signal_quality"`                     // 424: 2#GPS信号质量
	DeckPump1CoverSealPressure          float64 `json:"deck_pump_1_cover_seal_pressure"`         // 427: [JKT]1#甲板泵盖端封水压力(bar)
	DeckPump2CoverSealPressure          float64 `json:"deck_pump_2_cover_seal_pressure"`         // 428: [JKT]2#甲板泵盖端封水压力(bar)
	DeckPump1ShaftSealPressure          float64 `json:"deck_pump_1_shaft_seal_pressure"`         // 429: [JKT]1#甲板泵轴端封水压力(bar)
	DeckPump2ShaftSealPressure          float64 `json:"deck_pump_2_shaft_seal_pressure"`         // 430: [JKT]2#甲板泵轴端封水压力(bar)
	CutterDriveGateValveFlushPressure   float64 `json:"cutter_drive_gate_valve_flush_pressure"`  // 431: [JKT]绞刀驱动闸阀冲水压力(bar)
	CutterBearingFlushPressureJkt       float64 `json:"cutter_bearing_flush_pressure_jkt"`       // 432: [JKT]绞刀轴承冲水压力(bar)
	UnderwaterPumpCoverSealPressure     float64 `json:"underwater_pump_cover_seal_pressure"`     // 433: [JKT]水下泵盖端封水压力(bar)
	UnderwaterPumpShaftSealPressureJkt  float64 `json:"underwater_pump_shaft_seal_pressure_jkt"` // 434: [JKT]水下泵轴端封水压力(bar)
	DeckPump1GearboxOilTemperature      float64 `json:"deck_pump_1_gearbox_oil_temperature"`     // 435: [JKT]1#甲板泵齿轮箱滑油温度
	DeckPump1GearboxOilPressure         float64 `json:"deck_pump_1_gearbox_oil_pressure"`        // 436: [JKT]1#甲板泵齿轮箱滑油压力
	DeckPump2GearboxOilTemperature      float64 `json:"deck_pump_2_gearbox_oil_temperature"`     // 437: [JKT]2#甲板泵齿轮箱滑油温度
	DeckPump2GearboxOilPressure         float64 `json:"deck_pump_2_gearbox_oil_pressure"`        // 438: [JKT]2#甲板泵齿轮箱滑油压力
	CutterDriveGearboxOilTemperature    float64 `json:"cutter_drive_gearbox_oil_temperature"`    // 439: [JKT]绞刀驱动齿轮箱滑油温度
	CutterDriveGearboxOilPressure       float64 `json:"cutter_drive_gearbox_oil_pressure"`       // 440: [JKT]绞刀驱动齿轮箱滑油压力
	CutterDriveGearboxOilSaturation     float64 `json:"cutter_drive_gearbox_oil_saturation"`     // 441: [JKT]绞刀驱动齿轮箱滑油进水饱和度
	UnderwaterPumpGearboxOilTemperature float64 `json:"underwater_pump_gearbox_oil_temperature"` // 442: [JKT]水下泵齿轮箱滑油温度
	UnderwaterPumpGearboxOilPressure    float64 `json:"underwater_pump_gearbox_oil_pressure"`    // 443: [JKT]水下泵齿轮箱滑油压力
	UnderwaterPumpGearboxOilSaturation  float64 `json:"underwater_pump_gearbox_oil_saturation"`  // 444: [JKT]水下泵齿轮箱滑油进水饱和度
	FuelTank40Level                     float64 `json:"fuel_tank_40_level"`                      // 445: [BPJ]燃油舱40液位状态显示(m)
	MerFuelDailyTankLevel               float64 `json:"mer_fuel_daily_tank_level"`               // 453: [BC]MER燃油日用柜液位(m)
	FuelTank3Level                      float64 `json:"fuel_tank_3_level"`                       // 461: [SBCL]燃油舱3液位(m)
	LubricatingOilTank5Level            float64 `json:"lubricating_oil_tank_5_level"`            // 462: [SBCL]滑油储存舱5液位(m)
	HydraulicOilTank7Level              float64 `json:"hydraulic_oil_tank_7_level"`              // 463: [SBCL]液压油储存舱7液位(m)
	AuxiliaryFuelDailyTankLevel         float64 `json:"auxiliary_fuel_daily_tank_level"`         // 464: [SBCL]辅机舱燃油日用柜液位(m)
	FuelTank13Level                     float64 `json:"fuel_tank_13_level"`                      // 465: [SBCL]燃油舱13液位(m)
	FuelTank3ALevel                     float64 `json:"fuel_tank_3a_level"`                      // 466: [SBCL]燃油舱3A液位状态显示(m)
	FuelTank4Level                      float64 `json:"fuel_tank_4_level"`                       // 469: [SBCR]燃油舱4液位(m)
	SewageTank6Level                    float64 `json:"sewage_tank_6_level"`                     // 470: [SBCR]污水舱6液位(m)
	FreshwaterTank8Level                float64 `json:"freshwater_tank_8_level"`                 // 471: [SBCR]淡水舱8液位(m)
	DirtyOilTank11Level                 float64 `json:"dirty_oil_tank_11_level"`                 // 472: [SBCR]污油舱11液位(m)
	FuelTank12Level                     float64 `json:"fuel_tank_12_level"`                      // 473: [SBCR]燃油舱12液位(m)
	FreshwaterTank26Level               float64 `json:"freshwater_tank_26_level"`                // 474: [SBCR]淡水舱26液位(m)
	FuelTank4ALevel                     float64 `json:"fuel_tank_4a_level"`                      // 475: [SBCR]燃油舱4A液位显示状态(m)
}

// clients 用于存储所有活跃的WebSocket连接
var clients = make(map[*websocket.Conn]bool)

// mutex 用于在多goroutine环境下安全地访问clients映射
var mutex = &sync.Mutex{}

// handleConnections 处理WebSocket连接的函数
// 此函数将处理来自前端的连接请求，并启动与传感器的TCP通信
func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading to websocket: %v", err)
		return
	}
	defer ws.Close()

	mutex.Lock()
	clients[ws] = true
	mutex.Unlock()

	log.Println("websocket client connected")

	// 循环等待前端发送传感器地址
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("error reading websocket message: %v", err)
			mutex.Lock()
			delete(clients, ws)
			mutex.Unlock()
			break
		}

		// 收到的消息是传感器地址
		sensorAddr := string(msg)
		log.Printf("received sensor address: %s", sensorAddr)

		// 为每个地址启动一个独立的goroutine进行TCP通信和数据推送
		go handleSensorTCP(ws, sensorAddr)
	}
}

// handleSensorTCP 负责与单个传感器进行TCP通信，并将数据通过指定的WebSocket连接发送回前端
func handleSensorTCP(ws *websocket.Conn, addr string) {
	// 建立TCP连接
	tcpConn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		log.Printf("failed to connect to sensor at %s: %v", addr, err)
		// 可以选择向前端发送一条错误消息
		ws.WriteJSON(gin.H{"error": "Failed to connect to sensor"})
		return
	}
	defer tcpConn.Close()
	log.Printf("successfully connected to sensor at %s", addr)

	// 准备协议中定义的发送指令
	command := []byte{0x40, 0xFF, 0x00, 0x00, 0x0D, 0x0A}

	// 获取“华安龙”的配置，用于后续计算
	shipCfg := service.GetCfg("华安龙")

	// 使用 Ticker 每秒钟触发一次数据请求
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 1. 发送指令
		if _, err := tcpConn.Write(command); err != nil {
			log.Printf("error writing to tcp connection: %v", err)
			return // 写入失败，通常意味着连接已断开，退出goroutine
		}

		// 2. 读取和解析响应
		// 根据协议，先读取16字节的固定头部
		header := make([]byte, 16)
		if _, err := io.ReadFull(tcpConn, header); err != nil {
			log.Printf("error reading tcp header: %v", err)
			return
		}

		// 校验起始符
		if header[0] != 0x40 || header[1] != 0x01 {
			log.Println("invalid start of frame received")
			continue // 继续下一次循环
		}

		// 解析 DI 和 AI 数据的字节数（大端序）
		diLen := binary.BigEndian.Uint16(header[12:14])
		aiLen := binary.BigEndian.Uint16(header[14:16])

		// 计算AI数据点的数量（每个点4字节）
		aiPointCount := int(aiLen / 4)

		// 根据长度读取剩余的数据（DI数据 + AI数据 + 2字节校验和 + 2字节结束符）
		bodyLen := int(diLen) + int(aiLen) + 4
		body := make([]byte, bodyLen)
		if _, err := io.ReadFull(tcpConn, body); err != nil {
			log.Printf("error reading tcp body: %v", err)
			return
		}

		// 3. 提取AI数据并转换为浮点数
		// DI数据暂时忽略，直接跳到AI数据部分
		aiDataBytes := body[diLen : diLen+aiLen]
		aiFloats := make([]float32, aiPointCount)
		reader := bytes.NewReader(aiDataBytes)

		// 使用 binary.Read 读取所有浮点数（小端序）
		if err := binary.Read(reader, binary.LittleEndian, &aiFloats); err != nil {
			log.Printf("error parsing AI floats: %v", err)
			continue
		}

		// 4. 填充模型并计算预估真空度
		// 创建一个 model 实例用于存放解析后的数据
		dredgerData := &model.DredgerDataHl{}
		// 协议中的AI数据点索引从327开始，所以需要减去327来对应到数组索引
		// 更新了长度检查，以确保所有需要的字段都在范围内
		if len(aiFloats) > (475 - 327) {
			// --- 填充所有在 SensorData 结构体中定义的字段 ---
			dredgerData.LeftEarDraft = float64(aiFloats[327-327])
			dredgerData.UnderwaterPumpSuctionSealPressure = float64(aiFloats[328-327])
			dredgerData.UnderwaterPumpShaftSealPressure = float64(aiFloats[329-327])
			dredgerData.MudPump1ShaftSealPressure = float64(aiFloats[330-327])
			dredgerData.MudPump1SuctionSealPressure = float64(aiFloats[331-327])
			dredgerData.MudPump2SuctionSealPressure = float64(aiFloats[332-327])
			dredgerData.MudPump2ShaftSealPressure = float64(aiFloats[333-327])
			dredgerData.RightEarDraft = float64(aiFloats[334-327])
			dredgerData.LeftAnchorRodAngle = float64(aiFloats[335-327])
			dredgerData.RightAnchorRodAngle = float64(aiFloats[336-327])
			dredgerData.MudPump1Speed = float64(aiFloats[337-327])
			dredgerData.MudPump2Speed = float64(aiFloats[338-327])
			dredgerData.UnderwaterPumpSpeed = float64(aiFloats[339-327])
			dredgerData.FlowVelocity = float64(aiFloats[340-327])
			dredgerData.Density = float64(aiFloats[341-327])
			dredgerData.UnderwaterPumpMotorCurrent = float64(aiFloats[342-327])
			dredgerData.UnderwaterPumpMotorVoltage = float64(aiFloats[343-327])
			dredgerData.UnderwaterPumpTorque = float64(aiFloats[344-327])
			dredgerData.UnderwaterPumpMotorSpeed = float64(aiFloats[345-327])
			dredgerData.MudPump2DieselLoad = float64(aiFloats[346-327])
			dredgerData.MudPump2DieselSpeed = float64(aiFloats[347-327])
			dredgerData.MudPump1DieselLoad = float64(aiFloats[348-327])
			dredgerData.MudPump1DieselSpeed = float64(aiFloats[349-327])
			dredgerData.HydraulicPumpDieselLoad = float64(aiFloats[350-327])
			dredgerData.HydraulicPumpDieselSpeed = float64(aiFloats[351-327])
			dredgerData.GateValveFlushPressure = float64(aiFloats[352-327])
			dredgerData.CutterBearingFlushPressure = float64(aiFloats[353-327])
			dredgerData.TrolleyHydraulicCylinderPressure = float64(aiFloats[354-327])
			dredgerData.SteelPileHydraulicCylinderPressure = float64(aiFloats[355-327])
			dredgerData.GateValveSystemPressure = float64(aiFloats[356-327])
			dredgerData.RightTransversePressure = float64(aiFloats[357-327])
			dredgerData.LeftTransversePressure = float64(aiFloats[358-327])
			dredgerData.TrolleyTravel = float64(aiFloats[359-327])
			dredgerData.LeftTransverseSpeed = float64(aiFloats[360-327])
			dredgerData.RightTransverseSpeed = float64(aiFloats[361-327])
			dredgerData.CutterSpeed = float64(aiFloats[362-327])
			dredgerData.MudPump1DischargePressure = float64(aiFloats[363-327])
			dredgerData.MudPump2DischargePressure = float64(aiFloats[364-327])
			dredgerData.UnderwaterPumpDischargePressure = float64(aiFloats[365-327])
			dredgerData.UnderwaterPumpSuctionVacuum = float64(aiFloats[366-327])
			dredgerData.BridgeAngle = float64(aiFloats[367-327])
			dredgerData.CompassAngle = float64(aiFloats[368-327])
			dredgerData.Gps1X = float64(aiFloats[369-327])
			dredgerData.Gps1Y = float64(aiFloats[370-327])
			dredgerData.Gps1Heading = float64(aiFloats[371-327])
			dredgerData.Gps1Speed = float64(aiFloats[372-327])
			dredgerData.TideLevel = float64(aiFloats[373-327])
			dredgerData.WaterDensity = float64(aiFloats[374-327])
			dredgerData.FieldSlurryDensity = float64(aiFloats[375-327])
			dredgerData.TrimAngle = float64(aiFloats[376-327])
			dredgerData.PitchAngle = float64(aiFloats[377-327])
			dredgerData.CompassRadian = float64(aiFloats[378-327])
			dredgerData.Gps1Latitude = float64(aiFloats[379-327])
			dredgerData.Gps1Longitude = float64(aiFloats[380-327])
			dredgerData.EarDraft = float64(aiFloats[381-327])
			dredgerData.TransverseSpeed = float64(aiFloats[382-327])
			// 383: 绞刀转速 is a duplicate of 362, skipping
			dredgerData.HourlyOutputRate = float64(aiFloats[384-327])
			dredgerData.RotationRadius = float64(aiFloats[385-327])
			dredgerData.CutterX = float64(aiFloats[386-327])
			dredgerData.CutterY = float64(aiFloats[387-327])
			// Protocol index 388 'PreviousShiftProduction' does not exist in the model. Mapping to CurrentShiftOutput.
			dredgerData.CurrentShiftOutput = float64(aiFloats[388-327])
			dredgerData.CurrentShiftOutputRate = float64(aiFloats[389-327])
			dredgerData.OutletFlowVelocity = float64(aiFloats[390-327])
			dredgerData.LeftTransverseTorque = float64(aiFloats[391-327])
			dredgerData.CutterTorque = float64(aiFloats[392-327])
			dredgerData.Concentration = float64(aiFloats[393-327])
			dredgerData.FlowRate = float64(aiFloats[394-327])
			dredgerData.RightTransverseTorque = float64(aiFloats[395-327])
			dredgerData.LeftAnchorWinchSpeed = float64(aiFloats[396-327])
			dredgerData.LeftAnchorWinchTorque = float64(aiFloats[397-327])
			dredgerData.RightAnchorWinchSpeed = float64(aiFloats[398-327])
			dredgerData.RightAnchorWinchTorque = float64(aiFloats[399-327])
			dredgerData.LeftSwingWinchSpeed = float64(aiFloats[400-327])
			dredgerData.LeftSwingWinchTorque = float64(aiFloats[401-327])
			dredgerData.RightSwingWinchSpeed = float64(aiFloats[402-327])
			dredgerData.RightSwingWinchTorque = float64(aiFloats[403-327])
			dredgerData.BridgeWinchSpeed = float64(aiFloats[404-327])
			dredgerData.BridgeWinchTorque = float64(aiFloats[405-327])
			dredgerData.BridgeDepth = float64(aiFloats[406-327])
			dredgerData.TransverseDirection = int32(aiFloats[407-327])
			dredgerData.CutterCuttingAngle = float64(aiFloats[408-327])
			dredgerData.UnderwaterPumpPower = float64(aiFloats[409-327])
			dredgerData.MudPump1Power = float64(aiFloats[410-327])
			dredgerData.MudPump2Power = float64(aiFloats[411-327])
			dredgerData.UnderwaterPumpShaftPower = float64(aiFloats[412-327])
			dredgerData.MudPump1ShaftPower = float64(aiFloats[413-327])
			dredgerData.MudPump2ShaftPower = float64(aiFloats[414-327])
			dredgerData.UnderwaterPumpEfficiency = float64(aiFloats[415-327])
			dredgerData.MudPump1Efficiency = float64(aiFloats[416-327])
			dredgerData.MudPump2Efficiency = float64(aiFloats[417-327])
			dredgerData.PipelineAverageConcentration = float64(aiFloats[418-327])
			dredgerData.PipelineTotalDamping = float64(aiFloats[419-327])
			dredgerData.DensityForecast = float64(aiFloats[420-327])
			dredgerData.CuttingThickness = float64(aiFloats[421-327])
			dredgerData.ShipDirection = float64(aiFloats[422-327])
			dredgerData.Gps1SignalQuality = float64(aiFloats[423-327])
			dredgerData.Gps2SignalQuality = float64(aiFloats[424-327])
			dredgerData.DeckPump1CoverSealPressure = float64(aiFloats[427-327])
			dredgerData.DeckPump2CoverSealPressure = float64(aiFloats[428-327])
			dredgerData.DeckPump1ShaftSealPressure = float64(aiFloats[429-327])
			dredgerData.DeckPump2ShaftSealPressure = float64(aiFloats[430-327])
			dredgerData.CutterDriveGateValveFlushPressure = float64(aiFloats[431-327])
			dredgerData.CutterBearingFlushPressureJkt = float64(aiFloats[432-327])
			dredgerData.UnderwaterPumpCoverSealPressure = float64(aiFloats[433-327])
			dredgerData.UnderwaterPumpShaftSealPressureJkt = float64(aiFloats[434-327])
			dredgerData.DeckPump1GearboxOilTemperature = float64(aiFloats[435-327])
			dredgerData.DeckPump1GearboxOilPressure = float64(aiFloats[436-327])
			dredgerData.DeckPump2GearboxOilTemperature = float64(aiFloats[437-327])
			dredgerData.DeckPump2GearboxOilPressure = float64(aiFloats[438-327])
			dredgerData.CutterDriveGearboxOilTemperature = float64(aiFloats[439-327])
			dredgerData.CutterDriveGearboxOilPressure = float64(aiFloats[440-327])
			dredgerData.CutterDriveGearboxOilSaturation = float64(aiFloats[441-327])
			dredgerData.UnderwaterPumpGearboxOilTemperature = float64(aiFloats[442-327])
			dredgerData.UnderwaterPumpGearboxOilPressure = float64(aiFloats[443-327])
			dredgerData.UnderwaterPumpGearboxOilSaturation = float64(aiFloats[444-327])
			dredgerData.FuelTank40Level = float64(aiFloats[445-327])
			dredgerData.MerFuelDailyTankLevel = float64(aiFloats[453-327])
			dredgerData.FuelTank3Level = float64(aiFloats[461-327])
			dredgerData.LubricatingOilTank5Level = float64(aiFloats[462-327])
			dredgerData.HydraulicOilTank7Level = float64(aiFloats[463-327])
			dredgerData.AuxiliaryFuelDailyTankLevel = float64(aiFloats[464-327])
			dredgerData.FuelTank13Level = float64(aiFloats[465-327])
			dredgerData.FuelTank3ALevel = float64(aiFloats[466-327])
			dredgerData.FuelTank4Level = float64(aiFloats[469-327])
			dredgerData.SewageTank6Level = float64(aiFloats[470-327])
			dredgerData.FreshwaterTank8Level = float64(aiFloats[471-327])
			dredgerData.DirtyOilTank11Level = float64(aiFloats[472-327])
			dredgerData.FuelTank12Level = float64(aiFloats[473-327])
			dredgerData.FreshwaterTank26Level = float64(aiFloats[474-327])
			dredgerData.FuelTank4ALevel = float64(aiFloats[475-327])

		} else {
			log.Printf("received AI data length (%d) is not enough", len(aiFloats))
			continue
		}

		// 调用service计算预估真空度
		predictedVacuum := service.CalcVacuumKPaFromHL(dredgerData, shipCfg)

		// 5. 准备发送给前端的数据
		// 直接通过 dredgerData 构建 sensorData，确保字段一致
		sensorData := SensorData{
			PredictedVacuum:                     predictedVacuum,
			LeftEarDraft:                        dredgerData.LeftEarDraft,
			UnderwaterPumpSuctionSealPressure:   dredgerData.UnderwaterPumpSuctionSealPressure,
			UnderwaterPumpShaftSealPressure:     dredgerData.UnderwaterPumpShaftSealPressure,
			MudPump1ShaftSealPressure:           dredgerData.MudPump1ShaftSealPressure,
			MudPump1SuctionSealPressure:         dredgerData.MudPump1SuctionSealPressure,
			MudPump2SuctionSealPressure:         dredgerData.MudPump2SuctionSealPressure,
			MudPump2ShaftSealPressure:           dredgerData.MudPump2ShaftSealPressure,
			RightEarDraft:                       dredgerData.RightEarDraft,
			LeftAnchorRodAngle:                  dredgerData.LeftAnchorRodAngle,
			RightAnchorRodAngle:                 dredgerData.RightAnchorRodAngle,
			MudPump1Speed:                       dredgerData.MudPump1Speed,
			MudPump2Speed:                       dredgerData.MudPump2Speed,
			UnderwaterPumpSpeed:                 dredgerData.UnderwaterPumpSpeed,
			FlowVelocity:                        dredgerData.FlowVelocity,
			Density:                             dredgerData.Density,
			UnderwaterPumpMotorCurrent:          dredgerData.UnderwaterPumpMotorCurrent,
			UnderwaterPumpMotorVoltage:          dredgerData.UnderwaterPumpMotorVoltage,
			UnderwaterPumpTorque:                dredgerData.UnderwaterPumpTorque,
			UnderwaterPumpMotorSpeed:            dredgerData.UnderwaterPumpMotorSpeed,
			MudPump2DieselLoad:                  dredgerData.MudPump2DieselLoad,
			MudPump2DieselSpeed:                 dredgerData.MudPump2DieselSpeed,
			MudPump1DieselLoad:                  dredgerData.MudPump1DieselLoad,
			MudPump1DieselSpeed:                 dredgerData.MudPump1DieselSpeed,
			HydraulicPumpDieselLoad:             dredgerData.HydraulicPumpDieselLoad,
			HydraulicPumpDieselSpeed:            dredgerData.HydraulicPumpDieselSpeed,
			GateValveFlushPressure:              dredgerData.GateValveFlushPressure,
			CutterBearingFlushPressure:          dredgerData.CutterBearingFlushPressure,
			TrolleyHydraulicCylinderPressure:    dredgerData.TrolleyHydraulicCylinderPressure,
			SteelPileHydraulicCylinderPressure:  dredgerData.SteelPileHydraulicCylinderPressure,
			GateValveSystemPressure:             dredgerData.GateValveSystemPressure,
			RightTransversePressure:             dredgerData.RightTransversePressure,
			LeftTransversePressure:              dredgerData.LeftTransversePressure,
			TrolleyTravel:                       dredgerData.TrolleyTravel,
			LeftTransverseSpeed:                 dredgerData.LeftTransverseSpeed,
			RightTransverseSpeed:                dredgerData.RightTransverseSpeed,
			CutterSpeed:                         dredgerData.CutterSpeed,
			MudPump1DischargePressure:           dredgerData.MudPump1DischargePressure,
			MudPump2DischargePressure:           dredgerData.MudPump2DischargePressure,
			UnderwaterPumpDischargePressure:     dredgerData.UnderwaterPumpDischargePressure,
			UnderwaterPumpSuctionVacuum:         dredgerData.UnderwaterPumpSuctionVacuum,
			BridgeAngle:                         dredgerData.BridgeAngle,
			CompassAngle:                        dredgerData.CompassAngle,
			Gps1X:                               dredgerData.Gps1X,
			Gps1Y:                               dredgerData.Gps1Y,
			Gps1Heading:                         dredgerData.Gps1Heading,
			Gps1Speed:                           dredgerData.Gps1Speed,
			TideLevel:                           dredgerData.TideLevel,
			WaterDensity:                        dredgerData.WaterDensity,
			FieldSlurryDensity:                  dredgerData.FieldSlurryDensity,
			TrimAngle:                           dredgerData.TrimAngle,
			PitchAngle:                          dredgerData.PitchAngle,
			CompassRadian:                       dredgerData.CompassRadian,
			Gps1Latitude:                        dredgerData.Gps1Latitude,
			Gps1Longitude:                       dredgerData.Gps1Longitude,
			EarDraft:                            dredgerData.EarDraft,
			TransverseSpeed:                     dredgerData.TransverseSpeed,
			HourlyOutputRate:                    dredgerData.HourlyOutputRate,
			RotationRadius:                      dredgerData.RotationRadius,
			CutterX:                             dredgerData.CutterX,
			CutterY:                             dredgerData.CutterY,
			CurrentShiftOutput:                  dredgerData.CurrentShiftOutput,
			CurrentShiftOutputRate:              dredgerData.CurrentShiftOutputRate,
			OutletFlowVelocity:                  dredgerData.OutletFlowVelocity,
			LeftTransverseTorque:                dredgerData.LeftTransverseTorque,
			CutterTorque:                        dredgerData.CutterTorque,
			Concentration:                       dredgerData.Concentration,
			FlowRate:                            dredgerData.FlowRate,
			RightTransverseTorque:               dredgerData.RightTransverseTorque,
			LeftAnchorWinchSpeed:                dredgerData.LeftAnchorWinchSpeed,
			LeftAnchorWinchTorque:               dredgerData.LeftAnchorWinchTorque,
			RightAnchorWinchSpeed:               dredgerData.RightAnchorWinchSpeed,
			RightAnchorWinchTorque:              dredgerData.RightAnchorWinchTorque,
			LeftSwingWinchSpeed:                 dredgerData.LeftSwingWinchSpeed,
			LeftSwingWinchTorque:                dredgerData.LeftSwingWinchTorque,
			RightSwingWinchSpeed:                dredgerData.RightSwingWinchSpeed,
			RightSwingWinchTorque:               dredgerData.RightSwingWinchTorque,
			BridgeWinchSpeed:                    dredgerData.BridgeWinchSpeed,
			BridgeWinchTorque:                   dredgerData.BridgeWinchTorque,
			BridgeDepth:                         dredgerData.BridgeDepth,
			TransverseDirection:                 dredgerData.TransverseDirection,
			CutterCuttingAngle:                  dredgerData.CutterCuttingAngle,
			UnderwaterPumpPower:                 dredgerData.UnderwaterPumpPower,
			MudPump1Power:                       dredgerData.MudPump1Power,
			MudPump2Power:                       dredgerData.MudPump2Power,
			UnderwaterPumpShaftPower:            dredgerData.UnderwaterPumpShaftPower,
			MudPump1ShaftPower:                  dredgerData.MudPump1ShaftPower,
			MudPump2ShaftPower:                  dredgerData.MudPump2ShaftPower,
			UnderwaterPumpEfficiency:            dredgerData.UnderwaterPumpEfficiency,
			MudPump1Efficiency:                  dredgerData.MudPump1Efficiency,
			MudPump2Efficiency:                  dredgerData.MudPump2Efficiency,
			PipelineAverageConcentration:        dredgerData.PipelineAverageConcentration,
			PipelineTotalDamping:                dredgerData.PipelineTotalDamping,
			DensityForecast:                     dredgerData.DensityForecast,
			CuttingThickness:                    dredgerData.CuttingThickness,
			ShipDirection:                       dredgerData.ShipDirection,
			Gps1SignalQuality:                   dredgerData.Gps1SignalQuality,
			Gps2SignalQuality:                   dredgerData.Gps2SignalQuality,
			DeckPump1CoverSealPressure:          dredgerData.DeckPump1CoverSealPressure,
			DeckPump2CoverSealPressure:          dredgerData.DeckPump2CoverSealPressure,
			DeckPump1ShaftSealPressure:          dredgerData.DeckPump1ShaftSealPressure,
			DeckPump2ShaftSealPressure:          dredgerData.DeckPump2ShaftSealPressure,
			CutterDriveGateValveFlushPressure:   dredgerData.CutterDriveGateValveFlushPressure,
			CutterBearingFlushPressureJkt:       dredgerData.CutterBearingFlushPressureJkt,
			UnderwaterPumpCoverSealPressure:     dredgerData.UnderwaterPumpCoverSealPressure,
			UnderwaterPumpShaftSealPressureJkt:  dredgerData.UnderwaterPumpShaftSealPressureJkt,
			DeckPump1GearboxOilTemperature:      dredgerData.DeckPump1GearboxOilTemperature,
			DeckPump1GearboxOilPressure:         dredgerData.DeckPump1GearboxOilPressure,
			DeckPump2GearboxOilTemperature:      dredgerData.DeckPump2GearboxOilTemperature,
			DeckPump2GearboxOilPressure:         dredgerData.DeckPump2GearboxOilPressure,
			CutterDriveGearboxOilTemperature:    dredgerData.CutterDriveGearboxOilTemperature,
			CutterDriveGearboxOilPressure:       dredgerData.CutterDriveGearboxOilPressure,
			CutterDriveGearboxOilSaturation:     dredgerData.CutterDriveGearboxOilSaturation,
			UnderwaterPumpGearboxOilTemperature: dredgerData.UnderwaterPumpGearboxOilTemperature,
			UnderwaterPumpGearboxOilPressure:    dredgerData.UnderwaterPumpGearboxOilPressure,
			UnderwaterPumpGearboxOilSaturation:  dredgerData.UnderwaterPumpGearboxOilSaturation,
			FuelTank40Level:                     dredgerData.FuelTank40Level,
			MerFuelDailyTankLevel:               dredgerData.MerFuelDailyTankLevel,
			FuelTank3Level:                      dredgerData.FuelTank3Level,
			LubricatingOilTank5Level:            dredgerData.LubricatingOilTank5Level,
			HydraulicOilTank7Level:              dredgerData.HydraulicOilTank7Level,
			AuxiliaryFuelDailyTankLevel:         dredgerData.AuxiliaryFuelDailyTankLevel,
			FuelTank13Level:                     dredgerData.FuelTank13Level,
			FuelTank3ALevel:                     dredgerData.FuelTank3ALevel,
			FuelTank4Level:                      dredgerData.FuelTank4Level,
			SewageTank6Level:                    dredgerData.SewageTank6Level,
			FreshwaterTank8Level:                dredgerData.FreshwaterTank8Level,
			DirtyOilTank11Level:                 dredgerData.DirtyOilTank11Level,
			FuelTank12Level:                     dredgerData.FuelTank12Level,
			FreshwaterTank26Level:               dredgerData.FreshwaterTank26Level,
			FuelTank4ALevel:                     dredgerData.FuelTank4ALevel,
		}

		// 6. 通过WebSocket将数据发送到前端
		if err := ws.WriteJSON(sensorData); err != nil {
			log.Printf("error writing json to websocket: %v", err)
			// 写入失败，通常意味着WebSocket连接已关闭，应终止此goroutine
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
		api.POST("/demos/run", h.RunDemo)    // 上传+执行+返回新增文件
		api.POST("/files/open", h.OpenFile)  // Windows 打开文件
		api.GET("/files/serve", h.ServeFile) // 预览直链：/v1/files/serve?path=...
		api.GET("/demos/results/latest", h.GetLatestResults)
		api.POST("/files/open-location", h.OpenLocation)
		api.GET("/data/playback", h.GetPlaybackData)

		// WebSocket路由
		api.GET("/ws/sensor", func(c *gin.Context) {
			handleConnections(c.Writer, c.Request)
		})
	}

	return r
}
