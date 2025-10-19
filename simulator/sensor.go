package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"math/rand"
	"net"
	"time"
)

const (
	PORT     = ":4001"
	DI_COUNT = 320 // 协议中 DI 点位数量 (0 ~ 319)
	AI_COUNT = 157 // 协议中 AI 点位数量 (327 ~ 483)
)

func main() {
	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Fatal("监听失败:", err)
	}
	defer listener.Close()
	fmt.Println("华安龙传感器模拟器启动，监听端口 4001...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("接受连接失败:", err)
			continue
		}
		go handleConnection(conn)
	}
}

// handleConnection 现在会持续处理来自一个客户端的多个请求
func handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
	}()

	// *** FIX: 使用循环来持续处理请求 ***
	for {
		// 设置一个读取超时，例如5秒。如果5秒内没收到任何数据，
		// 就认为客户端已断开，然后关闭这个连接。
		err := conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			log.Println("设置读取超时失败:", err)
			return
		}

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			// 如果读取出错（例如，超时或连接被客户端关闭），就终止这个goroutine
			log.Println("读取数据失败 (客户端可能已断开):", err)
			return
		}

		// 收到数据后，取消超时设置
		err = conn.SetReadDeadline(time.Time{})
		if err != nil {
			log.Println("取消读取超时失败:", err)
			return
		}

		// 校验收到的指令
		expected := []byte{0x40, 0xFF, 0x00, 0x00, 0x0D, 0x0A}
		if n < len(expected) || !bytes.Equal(buf[:len(expected)], expected) {
			log.Printf("收到非法指令: % X", buf[:n])
			// 收到非法指令后不关闭连接，继续等待下一条
			continue
		}

		// 准备并发送响应数据
		response := prepareResponse()
		_, err = conn.Write(response)
		if err != nil {
			log.Println("发送数据失败:", err)
			return // 发送失败，通常意味着连接已断开
		}
		log.Println("✅ 已发送符合协议的完整响应数据")
	}
}

// prepareResponse 生成一条符合协议的模拟数据
// prepareResponse 生成一条符合协议的模拟数据
func prepareResponse() []byte {
	var response bytes.Buffer

	// 1. 起始符 (2 bytes)
	response.Write([]byte{0x40, 0x01})

	// 2. 备用字符 (10 bytes)
	response.Write(make([]byte, 10))

	// 3. DI 字节数 (2 bytes, 大端序)
	diByteLen := uint16((DI_COUNT + 7) / 8) // 320 bits = 40 bytes
	binary.Write(&response, binary.BigEndian, diByteLen)

	// 4. AI 字节数 (2 bytes, 大端序)
	aiByteLen := uint16(AI_COUNT * 4)
	binary.Write(&response, binary.BigEndian, aiByteLen)

	// 5. DI 数据 (40 bytes, 全0)
	diData := make([]byte, diByteLen)
	response.Write(diData)

	// 6. AI 数据
	aiData := make(map[int]float32)
	set := func(key int, baseValue float32) {
		aiData[key] = baseValue + (rand.Float32()-0.5)*baseValue*0.05 // ±5% 波动
	}

	// --- 按协议序号填充全部 157 个 AI 字段 ---
	set(327, 5.2)      // 左耳轴吃水 (m)
	set(328, 0.45)     // 水下泵吸入端封水压力 (bar)
	set(329, 0.5)      // 水下泵轴端封水压力
	set(330, 0.48)     // 1#泥泵轴端封水压力
	set(331, 0.42)     // 1#泥泵吸入端封水压力
	set(332, 0.43)     // 2#泥泵吸入端封水压力
	set(333, 0.49)     // 2#泥泵轴端封水压力
	set(334, 5.1)      // 右耳轴吃水
	set(335, 30.0)     // 左抛锚杆角度 (°)
	set(336, 32.0)     // 右抛锚杆角度
	set(337, 850.0)    // 1#泥泵转速 (rpm)
	set(338, 860.0)    // 2#泥泵转速
	set(339, 900.0)    // 水下泵转速
	set(340, 3.2)      // 流速 (m/s)
	set(341, 1.25)     // 密度 (t/m³)
	set(342, 420.0)    // 水下泵电机电流 (A)
	set(343, 690.0)    // 水下泵电机电压 (V)
	set(344, 1200.0)   // 水下泵扭矩 (kN·m)
	set(345, 895.0)    // 水下泵电机转速
	set(346, 75.0)     // 2#泥泵负荷 (%)
	set(347, 1800.0)   // 2#泥泵柴油机转速 (rpm)
	set(348, 72.0)     // 1#泥泵负荷
	set(349, 1780.0)   // 1#泥泵柴油机转速
	set(350, 68.0)     // 液压泵柴油机负荷
	set(351, 1750.0)   // 液压泵柴油机转速
	set(352, 0.6)      // 闸阀冲洗压力 (bar)
	set(353, 0.55)     // 绞刀轴承冲水压力
	set(354, 12.0)     // 台车液压油缸压力 (MPa)
	set(355, 15.0)     // 钢桩液压油缸压力
	set(356, 18.0)     // 闸阀系统压力
	set(357, 8.2)      // 右横移压力
	set(358, 8.0)      // 左横移压力
	set(359, 2.5)      // 台车行程 (m)
	set(360, 0.8)      // 左横移速度 (m/min)
	set(361, 0.78)     // 右横移速度
	set(362, 12.5)     // 绞刀转速 (rpm)
	set(363, 0.8)      // 1#泥泵排出压力 (MPa)
	set(364, 0.82)     // 2#泥泵排出压力
	set(365, 0.75)     // 水下泵排出压力
	set(366, -0.08)    // 水下泵吸入真空 (MPa)
	set(367, 35.5)     // 桥架角度 (°)
	set(368, 120.3)    // 罗经角度 (°)
	set(369, 121.4737) // GPS1_X (经度)
	set(370, 31.2304)  // GPS1_Y (纬度)
	set(371, 125.0)    // GPS1航向 (°)
	set(372, 5.2)      // GPS1航速 (kn)
	set(373, 1.2)      // 潮位 (m)
	set(374, 1.025)    // 水密度 (t/m³)
	set(375, 1.3)      // 现场泥浆比重
	set(376, 0.5)      // 横倾角度 (°)
	set(377, -0.3)     // 纵倾角度 (°)
	set(378, 2.1)      // 罗经弧度 (rad)
	set(379, 31.2304)  // GPS1_纬度
	set(380, 121.4737) // GPS1_经度
	set(381, 5.15)     // 耳轴吃水 (m)
	set(382, 0.79)     // 横移速度 (m/s)
	set(383, 12.6)     // 绞刀转速（重复）
	set(384, 1200.0)   // 小时产量率 (m³/h)
	set(385, 30.0)     // 旋转半径 (m)
	set(386, 121.48)   // 绞刀x (经度)
	set(387, 31.24)    // 绞刀y (纬度)
	set(388, 8500.0)   // 上一班组产量 (m³)
	set(389, 4200.0)   // 当前班产量
	set(390, 3.3)      // 出口流速 (m/s)
	set(391, 150.0)    // 左横移扭矩 (kN·m)
	set(392, 200.0)    // 绞刀扭矩
	set(393, 25.0)     // 浓度 (%)
	set(394, 3200.0)   // 流量 (m³/h)
	set(395, 148.0)    // 右横移扭矩
	set(396, 0.6)      // 左起锚绞车速度 (m/s)
	set(397, 80.0)     // 左起锚绞车扭矩
	set(398, 0.62)     // 右起锚绞车速度
	set(399, 82.0)     // 右起锚绞车扭矩
	set(400, 0.5)      // 左回转绞车速度
	set(401, 70.0)     // 左回转绞车扭矩
	set(402, 0.51)     // 右回转绞车速度
	set(403, 72.0)     // 右回转绞车扭矩
	set(404, 0.4)      // 起桥绞车速度
	set(405, 90.0)     // 起桥绞车扭矩
	set(406, 12.0)     // 桥架深度 (m)
	set(407, 1.0)      // 横移方向（1=左，-1=右）
	set(408, 45.0)     // 绞刀切削角 (°)
	set(409, 1800.0)   // 水下泵功率 (kW)
	set(410, 1500.0)   // 1#泥泵功率
	set(411, 1520.0)   // 2#泥泵功率
	set(412, 1750.0)   // 水下泵轴端驱动功率
	set(413, 1480.0)   // 1#泥泵轴端驱动功率
	set(414, 1500.0)   // 2#泥泵轴端驱动功率
	set(415, 85.0)     // 水下泵泵效 (%)
	set(416, 82.0)     // 1#泥泵泵效
	set(417, 83.0)     // 2#泥泵泵效
	set(418, 1.28)     // 管路平均浓度 (t/m³)
	set(419, 0.35)     // 管路总阻尼 (MPa)
	set(420, 1.26)     // 密度预报值
	set(421, 0.8)      // 切泥厚度 (m)
	set(422, 118.5)    // 船体方向 (°)
	set(423, 4.0)      // 1#GPS信号质量 (0~5)
	set(424, 3.8)      // 2#GPS信号质量

	// [JKT] 系列
	set(427, 0.44) // 1#甲板泵盖端封水压力
	set(428, 0.46) // 2#甲板泵盖端封水压力
	set(429, 0.51) // 1#甲板泵轴端封水压力
	set(430, 0.52) // 2#甲板泵轴端封水压力
	set(431, 0.6)  // 绞刀驱动闸阀冲水压力
	set(432, 0.55) // 绞刀轴承冲水压力
	set(433, 0.47) // 水下泵盖端封水压力
	set(434, 0.5)  // 水下泵轴端封水压力
	set(435, 65.0) // 1#甲板泵齿轮箱滑油温度 (°C)
	set(436, 0.35) // 1#甲板泵齿轮箱滑油压力 (MPa)
	set(437, 66.0) // 2#甲板泵齿轮箱滑油温度
	set(438, 0.36) // 2#甲板泵齿轮箱滑油压力
	set(439, 70.0) // 绞刀驱动齿轮箱滑油温度
	set(440, 0.4)  // 绞刀驱动齿轮箱滑油压力
	set(441, 15.0) // 绞刀驱动齿轮箱滑油进水饱和度 (%)
	set(442, 72.0) // 水下泵齿轮箱滑油温度
	set(443, 0.42) // 水下泵齿轮箱滑油压力
	set(444, 12.0) // 水下泵齿轮箱滑油进水饱和度

	// [BPJ] 燃油舱
	set(445, 3.2) // 燃油舱40液位状态显示 (m)

	// [BC] 燃油日用柜
	set(453, 2.1) // MER燃油日用柜液位 (m)

	// [SBCL]
	set(461, 4.5) // 燃油舱3液位
	set(462, 2.8) // 滑油储存舱5液位
	set(463, 3.0) // 液压油储存舱7液位
	set(464, 1.8) // 辅机舱燃油日用柜液位
	set(465, 5.0) // 燃油舱13液位
	set(466, 4.7) // 燃油舱3A液位状态显示

	// [SBCR]
	set(469, 4.3) // 燃油舱4液位
	set(470, 1.2) // 污水舱6液位
	set(471, 6.0) // 淡水舱8液位
	set(472, 0.9) // 污油舱11液位
	set(473, 4.1) // 燃油舱12液位
	set(474, 5.8) // 淡水舱26液位
	set(475, 4.4) // 燃油舱4A液位显示状态

	// 476 ~ 483 为备用，保持 0.0（map 中未设置，后续会自动补 0）

	// 将 AI 数据按序号 327~483 依次写入（小端序）
	var aiBuf bytes.Buffer
	for i := 327; i <= 483; i++ {
		val := aiData[i] // 未设置的 key 默认为 0.0
		binary.Write(&aiBuf, binary.LittleEndian, val)
	}
	response.Write(aiBuf.Bytes())

	// 7. 校验和（高字节=1，低字节=sum & 0xFF）
	data := response.Bytes()
	var sum uint16
	for _, b := range data {
		sum += uint16(b)
	}
	response.Write([]byte{0x01, byte(sum & 0xFF)})

	// 8. 结束符
	response.Write([]byte{0x0D, 0x0A})

	return response.Bytes()
}
