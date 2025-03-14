// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameDredgerDatum = "dredger_data"

// DredgerDatum 绞吸式挖泥船施工数据表
type DredgerDatum struct {
	ID                                int64   `gorm:"column:id;primaryKey;autoIncrement:true;comment:主键id" json:"id"`                                               // 主键id
	ShipName                          string  `gorm:"column:ship_name;not null;comment:船名" json:"ship_name"`                                                        // 船名
	RecordTime                        int64   `gorm:"column:record_time;comment:时间" json:"record_time"`                                                             // 时间
	LeftEarDraft                      float64 `gorm:"column:left_ear_draft;comment:左耳轴吃水" json:"left_ear_draft"`                                                    // 左耳轴吃水
	UnderwaterPumpSuctionSealPressure float64 `gorm:"column:underwater_pump_suction_seal_pressure;comment:水下泵吸入端封水压力" json:"underwater_pump_suction_seal_pressure"` // 水下泵吸入端封水压力
	UnderwaterPumpShaftSealPressure   float64 `gorm:"column:underwater_pump_shaft_seal_pressure;comment:水下泵轴端封水压力" json:"underwater_pump_shaft_seal_pressure"`      // 水下泵轴端封水压力
	RightEarDraft                     float64 `gorm:"column:right_ear_draft;comment:右耳轴吃水" json:"right_ear_draft"`                                                  // 右耳轴吃水
	UnderwaterPumpSpeed               float64 `gorm:"column:underwater_pump_speed;comment:水下泵转速" json:"underwater_pump_speed"`                                      // 水下泵转速
	FlowVelocity                      float64 `gorm:"column:flow_velocity;comment:流速" json:"flow_velocity"`                                                         // 流速
	Density                           float64 `gorm:"column:density;comment:密度" json:"density"`                                                                     // 密度
	UnderwaterPumpMotorCurrent        float64 `gorm:"column:underwater_pump_motor_current;comment:水下泵电机电流" json:"underwater_pump_motor_current"`                    // 水下泵电机电流
	TrolleyTravel                     float64 `gorm:"column:trolley_travel;comment:台车行程" json:"trolley_travel"`                                                     // 台车行程
	CutterSpeed                       float64 `gorm:"column:cutter_speed;comment:绞刀转速" json:"cutter_speed"`                                                         // 绞刀转速
	UnderwaterPumpSuctionVacuum       float64 `gorm:"column:underwater_pump_suction_vacuum;comment:水下泵吸入真空" json:"underwater_pump_suction_vacuum"`                  // 水下泵吸入真空
	BridgeAngle                       float64 `gorm:"column:bridge_angle;comment:桥架角度" json:"bridge_angle"`                                                         // 桥架角度
	CompassAngle                      float64 `gorm:"column:compass_angle;comment:罗经角度" json:"compass_angle"`                                                       // 罗经角度
	Gps1X                             float64 `gorm:"column:gps1_x;comment:GPS1_X" json:"gps1_x"`                                                                   // GPS1_X
	Gps1Y                             float64 `gorm:"column:gps1_y;comment:GPS1_Y" json:"gps1_y"`                                                                   // GPS1_Y
	Gps1Heading                       float64 `gorm:"column:gps1_heading;comment:GPS1航向" json:"gps1_heading"`                                                       // GPS1航向
	Gps1Speed                         float64 `gorm:"column:gps1_speed;comment:GPS1航速" json:"gps1_speed"`                                                           // GPS1航速
	TideLevel                         float64 `gorm:"column:tide_level;comment:潮位" json:"tide_level"`                                                               // 潮位
	WaterDensity                      float64 `gorm:"column:water_density;comment:水密度" json:"water_density"`                                                        // 水密度
	FieldSlurryDensity                float64 `gorm:"column:field_slurry_density;comment:现场泥浆比重" json:"field_slurry_density"`                                       // 现场泥浆比重
	CompassRadian                     float64 `gorm:"column:compass_radian;comment:罗经弧度" json:"compass_radian"`                                                     // 罗经弧度
	Gps1Latitude                      float64 `gorm:"column:gps1_latitude;comment:GPS1_纬度" json:"gps1_latitude"`                                                    // GPS1_纬度
	Gps1Longitude                     float64 `gorm:"column:gps1_longitude;comment:GPS1_经度" json:"gps1_longitude"`                                                  // GPS1_经度
	EarDraft                          float64 `gorm:"column:ear_draft;comment:耳轴吃水" json:"ear_draft"`                                                               // 耳轴吃水
	TransverseSpeed                   float64 `gorm:"column:transverse_speed;comment:横移速度" json:"transverse_speed"`                                                 // 横移速度
	HourlyOutputRate                  float64 `gorm:"column:hourly_output_rate;comment:小时产量率" json:"hourly_output_rate"`                                            // 小时产量率
	RotationRadius                    float64 `gorm:"column:rotation_radius;comment:旋转半径" json:"rotation_radius"`                                                   // 旋转半径
	CutterX                           float64 `gorm:"column:cutter_x;comment:绞刀x" json:"cutter_x"`                                                                  // 绞刀x
	CutterY                           float64 `gorm:"column:cutter_y;comment:绞刀y" json:"cutter_y"`                                                                  // 绞刀y
	OutletFlowVelocity                float64 `gorm:"column:outlet_flow_velocity;comment:出口流速" json:"outlet_flow_velocity"`                                         // 出口流速
	Concentration                     float64 `gorm:"column:concentration;comment:浓度" json:"concentration"`                                                         // 浓度
	FlowRate                          float64 `gorm:"column:flow_rate;comment:流量" json:"flow_rate"`                                                                 // 流量
	CutterSystemPressure              float64 `gorm:"column:cutter_system_pressure;comment:绞刀系统工作压力" json:"cutter_system_pressure"`                                 // 绞刀系统工作压力
	BridgeWinchPressure               float64 `gorm:"column:bridge_winch_pressure;comment:桥架绞车压力" json:"bridge_winch_pressure"`                                     // 桥架绞车压力
	VacuumReleaseValvePressure        float64 `gorm:"column:vacuum_release_valve_pressure;comment:真空释放阀压力" json:"vacuum_release_valve_pressure"`                    // 真空释放阀压力
	MainPilePressure                  float64 `gorm:"column:main_pile_pressure;comment:主钢桩工作压力" json:"main_pile_pressure"`                                          // 主钢桩工作压力
	Gps2X                             float64 `gorm:"column:gps2_x;comment:GPS2_X" json:"gps2_x"`                                                                   // GPS2_X
	Gps2Y                             float64 `gorm:"column:gps2_y;comment:GPS2_Y" json:"gps2_y"`                                                                   // GPS2_Y
	Gps2Heading                       float64 `gorm:"column:gps2_heading;comment:GPS2航向" json:"gps2_heading"`                                                       // GPS2航向
	Gps2Speed                         float64 `gorm:"column:gps2_speed;comment:GPS2航速" json:"gps2_speed"`                                                           // GPS2航速
	NaturalSoilDensity                float64 `gorm:"column:natural_soil_density;comment:天然土密度" json:"natural_soil_density"`                                        // 天然土密度
	Gps2Latitude                      float64 `gorm:"column:gps2_latitude;comment:GPS2_纬度" json:"gps2_latitude"`                                                    // GPS2_纬度
	Gps2Longitude                     float64 `gorm:"column:gps2_longitude;comment:GPS2_经度" json:"gps2_longitude"`                                                  // GPS2_经度
	LeftDensity                       float64 `gorm:"column:left_density;comment:左密度" json:"left_density"`                                                          // 左密度
	LeftFlowVelocity                  float64 `gorm:"column:left_flow_velocity;comment:左流速" json:"left_flow_velocity"`                                              // 左流速
	SlurryOutput                      float64 `gorm:"column:slurry_output;comment:泥浆产量" json:"slurry_output"`                                                       // 泥浆产量
	DrySoilOutput                     float64 `gorm:"column:dry_soil_output;comment:干土产量" json:"dry_soil_output"`                                                   // 干土产量
	DryTonOutput                      float64 `gorm:"column:dry_ton_output;comment:干吨土方量" json:"dry_ton_output"`                                                    // 干吨土方量
	OutputRate                        float64 `gorm:"column:output_rate;comment:产量率" json:"output_rate"`                                                            // 产量率
	CurrentShiftOutputRate            float64 `gorm:"column:current_shift_output_rate;comment:当前班产量率(" json:"current_shift_output_rate"`                            // 当前班产量率(
	CurrentShiftOutput                float64 `gorm:"column:current_shift_output;comment:当前班产量" json:"current_shift_output"`                                        // 当前班产量
	TransverseDistance                float64 `gorm:"column:transverse_distance;comment:横移距离" json:"transverse_distance"`                                           // 横移距离
	CurrentShift                      int32   `gorm:"column:current_shift;comment:当前班次" json:"current_shift"`                                                       // 当前班次
	TransverseAngle                   float64 `gorm:"column:transverse_angle;comment:横移角度" json:"transverse_angle"`                                                 // 横移角度
	DailyCumulativeOutput             float64 `gorm:"column:daily_cumulative_output;comment:今日累计产量" json:"daily_cumulative_output"`                                 // 今日累计产量
	PreviousDayOutput                 float64 `gorm:"column:previous_day_output;comment:昨日产量" json:"previous_day_output"`                                           // 昨日产量
	CutterDepth                       float64 `gorm:"column:cutter_depth;comment:绞刀深度" json:"cutter_depth"`                                                         // 绞刀深度
	TransverseWinchPressure           float64 `gorm:"column:transverse_winch_pressure;comment:横移绞车压力" json:"transverse_winch_pressure"`                             // 横移绞车压力
	IntermediatePressure              float64 `gorm:"column:intermediate_pressure;comment:水下泵与升压泵中间压力" json:"intermediate_pressure"`                                // 水下泵与升压泵中间压力
	BoosterPumpDischargePressure      float64 `gorm:"column:booster_pump_discharge_pressure;comment:升压泵排出压力" json:"booster_pump_discharge_pressure"`                // 升压泵排出压力
	BoosterPumpShaftSealPressure      float64 `gorm:"column:booster_pump_shaft_seal_pressure;comment:升压泵轴端封水压力" json:"booster_pump_shaft_seal_pressure"`            // 升压泵轴端封水压力
	BoosterPumpSuctionSealPressure    float64 `gorm:"column:booster_pump_suction_seal_pressure;comment:升压泵吸口端封水压力" json:"booster_pump_suction_seal_pressure"`       // 升压泵吸口端封水压力
	MudPipeDiameter                   float64 `gorm:"column:mud_pipe_diameter;comment:泥管直径" json:"mud_pipe_diameter"`                                               // 泥管直径
	SetDiggingDepth                   float64 `gorm:"column:set_digging_depth;comment:设定挖深" json:"set_digging_depth"`                                               // 设定挖深
	SetOverDiggingDepth               float64 `gorm:"column:set_over_digging_depth;comment:设定超深" json:"set_over_digging_depth"`                                     // 设定超深
	TideStation                       float64 `gorm:"column:tide_station;comment:潮位站" json:"tide_station"`                                                          // 潮位站
	ManualTideLevel                   float64 `gorm:"column:manual_tide_level;comment:手动输入潮位" json:"manual_tide_level"`                                             // 手动输入潮位
	TrimAngleSource                   float64 `gorm:"column:trim_angle_source;comment:横倾纵倾角度来源" json:"trim_angle_source"`                                           // 横倾纵倾角度来源
	LoadingSpeed                      float64 `gorm:"column:loading_speed;comment:装载速度" json:"loading_speed"`                                                       // 装载速度
	EarToBottomDistance               float64 `gorm:"column:ear_to_bottom_distance;comment:耳轴到船底垂直距离" json:"ear_to_bottom_distance"`                                // 耳轴到船底垂直距离
	AutoTideLevel                     float64 `gorm:"column:auto_tide_level;comment:自动读入的潮位" json:"auto_tide_level"`                                                // 自动读入的潮位
	CuttingAngle                      float64 `gorm:"column:cutting_angle;comment:切削角度" json:"cutting_angle"`                                                       // 切削角度
	TransverseDirection               int32   `gorm:"column:transverse_direction;comment:横移方向" json:"transverse_direction"`                                         // 横移方向
	MainPilePivotX                    float64 `gorm:"column:main_pile_pivot_x;comment:主桩支点x" json:"main_pile_pivot_x"`                                              // 主桩支点x
	MainPilePivotY                    float64 `gorm:"column:main_pile_pivot_y;comment:主桩支点y" json:"main_pile_pivot_y"`                                              // 主桩支点y
	BridgeWaterDepth                  float64 `gorm:"column:bridge_water_depth;comment:绞刀(桥架)水面深度" json:"bridge_water_depth"`                                       // 绞刀(桥架)水面深度
	LeftTransverseAnchorDropped       float64 `gorm:"column:left_transverse_anchor_dropped;comment:左横移锚下锚" json:"left_transverse_anchor_dropped"`                   // 左横移锚下锚
	RightTransverseAnchorDropped      float64 `gorm:"column:right_transverse_anchor_dropped;comment:右横移锚下锚" json:"right_transverse_anchor_dropped"`                 // 右横移锚下锚
	LeftTransverseAnchorX             float64 `gorm:"column:left_transverse_anchor_x;comment:左横移锚x" json:"left_transverse_anchor_x"`                                // 左横移锚x
	LeftTransverseAnchorY             float64 `gorm:"column:left_transverse_anchor_y;comment:左横移锚y" json:"left_transverse_anchor_y"`                                // 左横移锚y
	RightTransverseAnchorX            float64 `gorm:"column:right_transverse_anchor_x;comment:右横移锚x" json:"right_transverse_anchor_x"`                              // 右横移锚x
	RightTransverseAnchorY            float64 `gorm:"column:right_transverse_anchor_y;comment:右横移锚y" json:"right_transverse_anchor_y"`                              // 右横移锚y
	LeftTransverseWorklineAngle       float64 `gorm:"column:left_transverse_workline_angle;comment:左横移与工作线角度" json:"left_transverse_workline_angle"`                // 左横移与工作线角度
	RightTransverseWorklineAngle      float64 `gorm:"column:right_transverse_workline_angle;comment:右横移与工作线角度" json:"right_transverse_workline_angle"`              // 右横移与工作线角度
	Damping                           float64 `gorm:"column:damping;comment:阻尼" json:"damping"`                                                                     // 阻尼
	CuttingThickness                  float64 `gorm:"column:cutting_thickness;comment:切削厚度" json:"cutting_thickness"`                                               // 切削厚度
	AverageConcentration              float64 `gorm:"column:average_concentration;comment:平均浓度" json:"average_concentration"`                                       // 平均浓度
	CurrentWorklineDirection          float64 `gorm:"column:current_workline_direction;comment:当前工作线方向" json:"current_workline_direction"`                          // 当前工作线方向
	CurrentWorklineStartX             float64 `gorm:"column:current_workline_start_x;comment:当前工作线起点x" json:"current_workline_start_x"`                             // 当前工作线起点x
	CurrentWorklineStartY             float64 `gorm:"column:current_workline_start_y;comment:当前工作线起点y" json:"current_workline_start_y"`                             // 当前工作线起点y
	CurrentWorklineEndX               float64 `gorm:"column:current_workline_end_x;comment:当前工作线终点x" json:"current_workline_end_x"`                                 // 当前工作线终点x
	CurrentWorklineEndY               float64 `gorm:"column:current_workline_end_y;comment:当前工作线终点y" json:"current_workline_end_y"`                                 // 当前工作线终点y
	ShipStatus                        int32   `gorm:"column:ship_status;comment:船舶状态" json:"ship_status"`                                                           // 船舶状态
	LeftTransverseAnchorDeviation     float64 `gorm:"column:left_transverse_anchor_deviation;comment:左横移锚偏离中心线距离" json:"left_transverse_anchor_deviation"`          // 左横移锚偏离中心线距离
	RightTransverseAnchorDeviation    float64 `gorm:"column:right_transverse_anchor_deviation;comment:右横移锚偏离中心线距离" json:"right_transverse_anchor_deviation"`        // 右横移锚偏离中心线距离
	CutterDeviation                   float64 `gorm:"column:cutter_deviation;comment:绞刀偏离中心线距离" json:"cutter_deviation"`                                            // 绞刀偏离中心线距离
	MainPileDeviation                 float64 `gorm:"column:main_pile_deviation;comment:主桩偏离中心线距离" json:"main_pile_deviation"`                                      // 主桩偏离中心线距离
	EarX                              float64 `gorm:"column:ear_x;comment:耳轴x" json:"ear_x"`                                                                        // 耳轴x
	EarY                              float64 `gorm:"column:ear_y;comment:耳轴y" json:"ear_y"`                                                                        // 耳轴y
	PreviousCutterDepth               float64 `gorm:"column:previous_cutter_depth;comment:绞刀头点上一次的深度" json:"previous_cutter_depth"`                                 // 绞刀头点上一次的深度
	LeftBridgeWinchSpeed              float64 `gorm:"column:left_bridge_winch_speed;comment:左舷桥架绞车速度(接口计算值)" json:"left_bridge_winch_speed"`                        // 左舷桥架绞车速度(接口计算值)
	RightBridgeWinchSpeed             float64 `gorm:"column:right_bridge_winch_speed;comment:右舷桥架绞车速度(接口计算值)" json:"right_bridge_winch_speed"`                      // 右舷桥架绞车速度(接口计算值)
	BridgeSpeed                       float64 `gorm:"column:bridge_speed;comment:桥架速度(接口计算值)" json:"bridge_speed"`                                                  // 桥架速度(接口计算值)
	LeftBridgeWinchSpeed2             float64 `gorm:"column:left_bridge_winch_speed2;comment:左舷桥架绞车速度2(接口计算值)" json:"left_bridge_winch_speed2"`                     // 左舷桥架绞车速度2(接口计算值)
	RightBridgeWinchSpeed2            float64 `gorm:"column:right_bridge_winch_speed2;comment:右舷桥架绞车速度2(接口计算值)" json:"right_bridge_winch_speed2"`                   // 右舷桥架绞车速度2(接口计算值)
	WindDirection                     float64 `gorm:"column:wind_direction;comment:风向(滤波后)" json:"wind_direction"`                                                  // 风向(滤波后)
	WindSpeed                         float64 `gorm:"column:wind_speed;comment:风速(滤波后)" json:"wind_speed"`                                                          // 风速(滤波后)
	CurrentWorklineDirectionDpm       float64 `gorm:"column:current_workline_direction_dpm;comment:当前工作线方向(自DPM)" json:"current_workline_direction_dpm"`            // 当前工作线方向(自DPM)
	CurrentWorklineStartXDpm          float64 `gorm:"column:current_workline_start_x_dpm;comment:当前工作线起点x(自DPM)" json:"current_workline_start_x_dpm"`               // 当前工作线起点x(自DPM)
	CurrentWorklineStartYDpm          float64 `gorm:"column:current_workline_start_y_dpm;comment:当前工作线起点y(自DPM)" json:"current_workline_start_y_dpm"`               // 当前工作线起点y(自DPM)
	CurrentWorklineEndXDpm            float64 `gorm:"column:current_workline_end_x_dpm;comment:当前工作线终点x(自DPM)" json:"current_workline_end_x_dpm"`                   // 当前工作线终点x(自DPM)
	CurrentWorklineEndYDpm            float64 `gorm:"column:current_workline_end_y_dpm;comment:当前工作线终点y(自DPM)" json:"current_workline_end_y_dpm"`                   // 当前工作线终点y(自DPM)
	PreviousCutterDepthDpm            float64 `gorm:"column:previous_cutter_depth_dpm;comment:绞刀头点上一次的深度(自DPM)" json:"previous_cutter_depth_dpm"`                   // 绞刀头点上一次的深度(自DPM)
	ShipDeviationAngle                float64 `gorm:"column:ship_deviation_angle;comment:船体偏移工作线角度" json:"ship_deviation_angle"`                                    // 船体偏移工作线角度
}

// TableName DredgerDatum's table name
func (*DredgerDatum) TableName() string {
	return TableNameDredgerDatum
}
