# 大乐透和双色球选号工具

一个基于Go语言开发的大乐透和双色球选号工具，支持历史数据获取、AI分析预测和选号生成，支持多平台部署（armv7/arm64/x86_64）。

## 功能特性

- **历史数据爬取**: 从网络抓取大乐透和双色球历史开奖数据
- **数据统计分析**: 冷热号分析、趋势分析、遗漏值统计
- **AI智能预测**: 支持OpenAI格式API，智能分析预测
- **多种预测策略**: 随机选号、热号策略、冷号策略、冷热混合、AI智能
- **命令行工具**: 完整的CLI命令，支持数据抓取、分析、预测
- **多平台支持**: 支持Linux/macOS/Windows，支持armv7/arm64/x86_64架构
- **Docker部署**: 提供Docker镜像和docker-compose配置

## 快速开始

### 1. 配置

复制配置文件并修改：

```bash
cp config/config.yaml config/config.local.yaml
# 编辑 config.local.yaml 填入你的配置
```

配置文件示例：

```yaml
app:
  name: lottery-tool
  version: 1.0.0

database:
  path: ./data/lottery.db

scraper:
  enabled: true
  interval: 86400
  dlt_url: "https://example.com/dlt-api"  # 大乐透数据源
  ssq_url: "https://example.com/ssq-api"  # 双色球数据源

ai:
  enabled: true
  api_url: "https://api.openai.com"       # AI API地址
  api_key: "your-api-key"                 # API密钥
  model: "gpt-3.5-turbo"                  # 模型名称
  timeout: 30

server:
  enabled: false
  port: 8080
  host: 0.0.0.0
```

### 2. 编译

#### 本地编译

```bash
# 编译当前平台版本
make build

# 运行测试
make test
```

#### 多平台交叉编译

```bash
# 编译所有支持的平台
make build-all

# 或单独编译特定平台
make build-linux-amd64    # Linux x86_64
make build-linux-arm64    # Linux ARM64
make build-linux-armv7    # Linux ARMv7
make build-darwin-amd64   # macOS Intel
make build-darwin-arm64   # macOS Apple Silicon
make build-windows-amd64  # Windows x86_64

# 或使用脚本构建
./deploy/scripts/build.sh
```

### 3. Docker部署

#### 使用Docker构建

```bash
# 构建镜像（支持多平台）
docker build -t lottery-tool:latest -f deploy/docker/Dockerfile .

# 或使用docker-compose
cd deploy/docker
docker-compose up -d
```

#### 使用Buildx构建多平台镜像

```bash
# 创建buildx构建器
docker buildx create --use

# 构建多平台镜像
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t lottery-tool:latest -f deploy/docker/Dockerfile --push .
```

## GitHub Actions 自动构建

项目提供了三个 GitHub Actions 工作流，支持手动触发多平台构建和发布：

### 1. 构建工作流 (`build.yml`)

手动触发构建指定平台的二进制文件：

1. 进入 GitHub 仓库的 **Actions** 页面
2. 选择 **构建** 工作流
3. 点击 **Run workflow**
4. 填写参数：
   - `version`: 版本号，如 `1.0.0`
   - `build_binaries`: 是否构建二进制文件
   - `build_docker`: 是否构建 Docker 镜像
5. 点击 **Run workflow** 开始构建

### 2. Docker 构建工作流 (`docker.yml`)

构建多平台 Docker 镜像（amd64 / arm64 / armv7）：

1. 进入 **Actions** → **Docker 构建**
2. 点击 **Run workflow**
3. 填写参数：
   - `version`: 版本号
   - `push`: 是否推送到 Docker Hub
4. 执行构建

**Docker Hub 推送配置**：需要在仓库 Settings → Secrets 中配置：
- `DOCKER_USERNAME`: Docker Hub 用户名
- `DOCKER_PASSWORD`: Docker Hub 密码或 Token

### 3. 发布工作流 (`release.yml`)

构建所有平台二进制文件并创建 GitHub Release：

1. 进入 **Actions** → **发布 Release**
2. 点击 **Run workflow**
3. 填写参数：
   - `version`: 版本号，如 `v1.0.0`
   - `release_notes`: 发布说明
4. 执行后自动创建 Release，包含所有平台的二进制包和校验和

### 构建产物

| 平台 | 架构 | 产物 |
|------|------|------|
| Linux | amd64 | `lottery-tool-linux-amd64` |
| Linux | arm64 | `lottery-tool-linux-arm64` |
| Linux | armv7 | `lottery-tool-linux-armv7` |
| Linux | armv6 | `lottery-tool-linux-armv6` |
| macOS | amd64 | `lottery-tool-darwin-amd64` |
| macOS | arm64 | `lottery-tool-darwin-arm64` |
| Windows | amd64 | `lottery-tool-windows-amd64.exe` |

## 使用指南

### CLI命令

```bash
# 查看帮助
./lottery-cli --help

# 查看版本
./lottery-cli version

# 抓取历史数据
./lottery-cli fetch dlt -l 100    # 抓取大乐透100期数据
./lottery-cli fetch ssq -l 100    # 抓取双色球100期数据

# 列出历史数据
./lottery-cli list dlt -l 20      # 显示大乐透最近20期
./lottery-cli list ssq -l 20      # 显示双色球最近20期

# 数据分析
./lottery-cli analyze dlt         # 分析大乐透数据
./lottery-cli analyze ssq         # 分析双色球数据

# 生成预测
./lottery-cli predict dlt                    # 随机策略预测大乐透
./lottery-cli predict dlt -s hot             # 热号策略
./lottery-cli predict dlt -s cold            # 冷号策略
./lottery-cli predict dlt -s mix             # 冷热混合策略
./lottery-cli predict dlt -s ai -a           # AI智能预测
./lottery-cli predict dlt -n 10              # 生成10组预测
```

### 预测策略说明

- **random**: 完全随机选号
- **hot**: 优先选择出现频率高的热号
- **cold**: 优先选择出现频率低的冷号
- **mix**: 60%热号 + 40%冷号的混合策略
- **ai**: 使用AI进行智能分析预测（需要配置AI API）

## 项目结构

```
lottery-tool/
├── cmd/                      # 应用入口
│   ├── cli/                 # 命令行工具
│   │   ├── main.go
│   │   └── commands.go
│   └── server/              # API服务（预留）
├── internal/                 # 内部包
│   ├── scraper/             # 爬虫模块
│   ├── storage/             # 数据存储（SQLite）
│   ├── analyzer/            # 数据分析
│   ├── ai/                  # AI分析
│   ├── predictor/           # 预测生成
│   └── config/              # 配置管理
├── pkg/                      # 公共包
│   ├── types/               # 数据类型
│   └── utils/               # 工具函数
├── deploy/                   # 部署相关
│   ├── docker/              # Docker配置
│   └── scripts/             # 构建脚本
├── data/                     # 数据存储目录
├── config/                   # 配置文件
├── Makefile                 # 构建脚本
├── go.mod
└── README.md
```

## 技术栈

- **Go 1.21+**: 编程语言
- **SQLite**: 嵌入式数据库
- **goquery**: HTML解析
- **cobra**: 命令行框架
- **viper**: 配置管理
- **Docker**: 容器化部署

## 支持的彩票类型

### 大乐透 (DLT)
- 红球：1-35，选5个
- 蓝球：1-12，选2个

### 双色球 (SSQ)
- 红球：1-33，选6个
- 蓝球：1-16，选1个

## 开发计划

- [x] 完成爬虫模块实现
- [x] 完成数据分析模块
- [x] 完成AI预测模块
- [x] 实现预测生成策略
- [x] 添加CLI子命令
- [ ] 实现REST API服务
- [ ] 添加Web管理界面
- [ ] 完善单元测试
- [ ] 支持更多彩票类型

## 注意事项

1. **数据源配置**: 爬虫模块需要配置有效的数据源URL，可以从彩票官方网站或第三方数据服务获取
2. **AI配置**: AI预测功能需要配置有效的OpenAI格式API密钥
3. **数据存储**: 默认使用SQLite数据库，数据存储在`data/lottery.db`
4. **随机性**: 彩票中奖是随机事件，本工具仅供娱乐和学习使用

## 许可证

MIT License

## 免责声明

本工具仅供学习和娱乐使用，不保证预测准确性。彩票有风险，投注需谨慎。请理性购彩，量力而行。
