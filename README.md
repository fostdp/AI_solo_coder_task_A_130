# 古代都江堰岁修工艺仿真与河床演变分析系统

## 项目概述

本系统是为水利史研究团队开发的全栈应用，用于模拟和分析古代都江堰的岁修工艺与河床演变过程。系统结合了三维可视化、离散元法仿真、水沙模型预测和实时数据监控等功能。

## 技术架构

### 后端技术栈
- **语言**: Go 1.21+
- **Web框架**: Gin v1.9.1
- **数据库**: PostgreSQL + TimescaleDB
- **数据库驱动**: pgx/v5 v5.5.0
- **消息队列**: MQTT (Eclipse Paho)
- **实时通信**: WebSocket (Gorilla)
- **科学计算**: Gonum v0.14.0

### 前端技术栈
- **3D渲染**: Three.js v0.158.0
- **图表库**: Chart.js v4.4.0
- **UI**: 原生HTML5 + CSS3 + JavaScript
- **2D绘图**: Canvas API

### 核心功能模块

#### 1. 数据采集与存储
- 每1小时模拟传感器上报水位、流量、含沙量、河床高程
- 8个监测站点（内江3个、外江2个、飞沙堰2个、人字堤1个）
- TimescaleDB时序数据库优化存储
- 连续聚合视图提供快速统计查询

#### 2. 岁修工艺仿真
- **杩槎截流仿真**: 基于离散元法模拟杩槎结构的截流过程
- **竹笼装石仿真**: 离散元法模拟石块填充竹笼的物理过程
- 碰撞检测与响应、重力、摩擦等物理效果

#### 3. 河床演变分析
- 基于水沙模型的河床冲淤计算
- 未来10年河床高程预测
- 季节性变化和流量变化因素考虑
- 动态高程图可视化

#### 4. 三维可视化
- 都江堰渠首三维地形模型（鱼嘴、飞沙堰、宝瓶口等）
- 水流粒子动画系统（5000+粒子，5条流路）
- 水利工程结构3D模型
- 卧铁标记与监测站点可视化

#### 5. 告警系统
- 河床淤积超过卧铁高程自动触发预警
- 三级告警级别（严重/警告/通知）
- MQTT消息推送
- 数据库触发器自动创建告警

## 目录结构

```
AI_solo_coder_task_A_130/
├── backend/                          # Go后端服务
│   ├── cmd/
│   │   └── server/
│   │       └── main.go               # 主服务入口
│   ├── pkg/
│   │   ├── api/
│   │   │   └── handlers.go           # HTTP API处理器
│   │   ├── models/
│   │   │   ├── models.go             # 数据模型定义
│   │   │   └── database.go           # 数据库操作
│   │   ├── mqtt/
│   │   │   └── mqtt_client.go        # MQTT客户端
│   │   └── simulation/
│   │       ├── bed_evolution.go      # 河床演变模型
│   │       └── dem_simulation.go     # 离散元法仿真
│   ├── .env                          # 环境变量
│   └── go.mod                        # Go依赖
├── frontend/                         # 前端应用
│   ├── index.html                    # 主页面
│   ├── css/
│   │   └── style.css                 # 样式文件
│   └── js/
│       ├── config.js                 # 配置文件
│       ├── api.js                    # API封装
│       ├── utils.js                  # 工具函数
│       ├── main.js                   # 主应用逻辑
│       ├── visualization/
│       │   ├── three-scene.js        # Three.js场景管理
│       │   ├── particles.js          # 水流粒子系统
│       │   ├── terrain.js            # 地形生成器
│       │   └── structures.js         # 结构生成器
│       └── simulation/
│           ├── macha.js              # 杩槎仿真渲染
│           ├── bamboo.js             # 竹笼仿真渲染
│           └── dem.js                # 动态高程图渲染
├── scripts/                          # 脚本文件
│   ├── init_timescaledb.sql          # 数据库初始化脚本
│   └── hydrology_simulator.go        # 水文数据模拟器
├── config/
│   └── config.yaml                   # 配置文件
└── README.md                         # 项目文档
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- PostgreSQL 14+
- TimescaleDB 2.13+
- MQTT Broker (如 Mosquitto)
- 现代浏览器（支持WebGL）

### 2. 数据库初始化

```sql
-- 创建数据库
CREATE DATABASE dujiangyan;

-- 连接数据库
\c dujiangyan

-- 执行初始化脚本
\i scripts/init_timescaledb.sql
```

### 3. 后端配置

编辑 `backend/.env` 文件：

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=dujiangyan

MQTT_BROKER=tcp://localhost:1883
MQTT_CLIENT_ID=dujiangyan_backend
MQTT_TOPIC_ALERT=dujiangyan/alerts
MQTT_TOPIC_HYDROLOGY=dujiangyan/hydrology

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
```

### 4. 编译运行后端

```bash
cd backend
go mod download
go build -o server cmd/server/main.go
./server
```

### 5. 运行水文模拟器

```bash
cd scripts
go run hydrology_simulator.go -api http://localhost:8080/api/v1 -interval 1s -speed 3600
```

参数说明：
- `-api`: 后端API地址
- `-interval`: 数据上报间隔（真实时间）
- `-speed`: 时间加速倍数（3600表示1秒=1小时）
- `-historical`: 预生成历史数据天数

### 6. 启动MQTT Broker（可选）

```bash
# 使用Docker启动Mosquitto
docker run -d -p 1883:1883 eclipse-mosquitto:latest
```

### 7. 访问前端

直接在浏览器中打开 `frontend/index.html`，或使用静态文件服务器：

```bash
# 使用Python启动简单服务器
cd frontend
python -m http.server 8081

# 或使用Node.js
npx http-server frontend -p 8081
```

访问 `http://localhost:8081` 即可使用系统。

## API接口文档

### 水文数据接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/hydrology/data` | 上报水文数据 |
| GET | `/api/v1/hydrology/data/:station_id` | 查询历史数据 |
| GET | `/api/v1/hydrology/data/latest/:station_id` | 获取最新数据 |
| GET | `/api/v1/hydrology/data/all` | 获取所有站点最新数据 |
| GET | `/api/v1/hydrology/stats/daily/:station_id` | 获取日统计数据 |
| GET | `/api/v1/hydrology/stations` | 获取监测站点列表 |

### 预测与仿真接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/prediction/bed-evolution/:station_id` | 运行河床演变预测 |
| GET | `/api/v1/prediction/bed-evolution/:station_id` | 获取预测结果 |
| POST | `/api/v1/simulation/bamboo-cage` | 运行竹笼装石仿真 |
| POST | `/api/v1/simulation/macha-interception` | 运行杩槎截流仿真 |
| GET | `/api/v1/dem-grid` | 获取DEM高程网格 |

### 告警接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/alerts` | 获取告警列表 |
| POST | `/api/v1/alerts/:id/acknowledge` | 确认告警 |

### WebSocket接口

- `ws://host:port/api/v1/ws/realtime` - 实时数据推送

## 核心算法说明

### 1. 离散元法 (DEM) 仿真

```go
// 核心物理更新
func (sim *DEMSimulation) Step(dt float64) {
    // 1. 应用重力和外力
    // 2. 检测碰撞
    // 3. 冲量法解析碰撞
    // 4. 速度和位置更新
    // 5. 摩擦和阻尼应用
}
```

### 2. 水沙输运模型

```go
// 输沙率公式 Qs = K * Q^n * S^m
func (model *SedimentTransportModel) CalculateSedimentRate(
    flowRate, slope, sedimentConcentration float64,
) float64 {
    return model.K * 
           math.Pow(flowRate, model.ExponentFlow) * 
           math.Pow(slope, model.ExponentSlope) *
           sedimentConcentration
}
```

### 3. 河床演变预测

基于历史数据统计分析 + 过程模型：
- 年均淤积/冲刷速率计算
- 季节性变化因子（汛期/枯水期）
- 流量变化影响系数
- 随机扰动项

## 功能模块说明

### 总览页面
- 都江堰渠首三维全景缩略图
- 8个站点实时水文参数展示
- 监测站点分布图（Canvas绘制）
- 河床演变趋势图
- 卧铁与河床高程对比图

### 三维模型页面
- 完整的都江堰渠首3D模型
- 图层控制（地形、水流、结构、卧铁、站点）
- 水流粒子动画（可调节粒子数量和水位放大倍数）
- 预设视图（总览、内江、外江、宝瓶口、飞沙堰）
- 鼠标悬停信息面板

### 河床演变页面
- 站点选择和预测年限设置
- 河床高程预测曲线
- 冲淤速率柱状图
- 动态高程图（时间轴拖动）
- 统计指标卡片（年均淤积、年均冲刷、10年后高程、风险等级）

### 岁修仿真页面
- **杩槎截流**：2D Canvas物理仿真，实时截流效率曲线
- **竹笼装石**：离散元法石块填充仿真，稳定性分布图表
- **岁修记录**：历史岁修数据表格

### 告警中心
- 告警列表（可按级别和确认状态筛选）
- 告警详情（河床高程、卧铁高程、超覆高度）
- 告警确认功能
- 实时告警弹窗通知

### 数据监控页面
- 多站点、多时间范围数据查询
- 水位、流量、含沙量、河床高程趋势图
- 实时数据自动更新

## 数据格式说明

### 水文数据结构
```json
{
    "station_id": "NEIJ-001",
    "water_level": 728.5,
    "flow_rate": 350.0,
    "sediment_concentration": 0.8,
    "bed_elevation": 726.5,
    "timestamp": "2024-01-01T00:00:00Z"
}
```

### 告警级别说明
- **CRITICAL (严重)**: 河床高程 > 卧铁高程 + 1.0m
- **WARNING (警告)**: 河床高程 > 卧铁高程 + 0.5m
- **NOTICE (通知)**: 河床高程 > 卧铁高程

## 配置说明

### 仿真参数 (config.yaml)
```yaml
simulation:
  dem:
    gravity: 9.81
    restitution: 0.3
    friction: 0.98
    viscosity: 0.01
  bed_evolution:
    K: 0.001
    exponent_flow: 2.0
    exponent_slope: 1.5
    porosity: 0.4
    bulk_density: 2650.0
```

## 故障排除

### 后端无法连接数据库
- 检查PostgreSQL和TimescaleDB是否正常运行
- 确认 `.env` 中的数据库配置正确
- 确认数据库已执行初始化脚本

### MQTT连接失败
- 检查MQTT Broker是否运行
- 确认端口1883是否开放
- 系统支持无MQTT运行（告警仅存储不推送）

### 前端3D场景不显示
- 确认浏览器支持WebGL
- 检查控制台是否有错误信息
- 尝试刷新页面重新加载

### 粒子动画不流畅
- 降低粒子数量（2000以下）
- 关闭其他浏览器标签释放资源
- 使用性能更好的显卡

## 开发说明

### 后端开发
```bash
cd backend
go run cmd/server/main.go
```

### 前端开发
前端使用纯HTML/JS/CSS，无需构建工具，直接修改文件即可。

### 数据库扩展
如需添加新的监测指标：
1. 在 `init_timescaledb.sql` 中添加对应字段
2. 更新 `models.go` 中的数据结构
3. 在 `database.go` 添加相应的查询方法

## 许可证

本项目仅供水利史研究使用。

## 参考文献

1. 《都江堰水利工程史》
2. 《离散元法及其在岩土工程中的应用》
3. 《泥沙运动力学》
4. TimescaleDB官方文档
5. Three.js官方文档
