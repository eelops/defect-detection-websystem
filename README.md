# 缺陷检测管理系统

一个基于 **React + Ant Design + Go + SQLite** 的缺陷检测管理系统，面向制造业缺陷登记、处理闭环、基础数据维护和审计追踪场景。当前版本已支持账号登录、角色权限、缺陷流转、基础数据管理、Excel/PDF 报表导出、操作日志、严重缺陷提醒和外部模型推送。

## 功能特性

- 数据看板：缺陷总数、等级统计、类型分布、近 7 天趋势
- 缺陷管理：新增缺陷、列表筛选、详情查看、状态更新、删除记录
- 缺陷闭环流程：指派、处理、提交验收、验收通过/退回、关闭，支持流转记录查看
- 基础数据管理：产线、设备、缺陷类型维护，并可在缺陷录入时关联主数据
- 报警通知：严重缺陷顶部告警条、顶部铃铛提醒、浏览器桌面通知、声音提醒、创建后立即弹出提醒
- 外部推送：第三方检测模型通过 `POST /api/external/report` 自动上报缺陷
- 邮件通知：创建严重缺陷后自动向管理员邮箱发送告警邮件
- 用户登录：账号密码登录、退出登录、本人修改密码
- 角色权限：管理员、质检员、查看员 3 类角色访问控制
- 用户管理：管理员可新增用户、修改角色、启用/禁用账号、重置密码
- 操作日志：管理员可查看关键业务操作和安全审计记录
- 报表导出：缺陷列表支持 Excel 导出，缺陷详情支持 PDF 导出

## 技术栈

- 前端：React 19、Ant Design 5、Axios
- 后端：Go、net/http、modernc.org/sqlite
- 数据库：SQLite
- 辅助脚本：Python 3（仅用于运行 `test_external.py`）
- 部署：Docker / Docker Compose

## 项目结构

```text
.
├─ backend/                  # Go 后端服务
│  ├─ main.go                # 后端主程序与核心业务接口
│  ├─ masterdata.go          # 基础数据管理、校验与导出相关逻辑
│  ├─ main_test.go           # Go 自动化测试
│  ├─ go.mod                 # Go 依赖配置
│  ├─ defects.db             # SQLite 数据库文件（运行后生成/更新，建议忽略）
│  ├─ uploads/               # 上传图片目录（运行后生成/写入，建议忽略）
│  ├─ Dockerfile             # 后端镜像构建文件
│  └─ .env                   # 后端环境变量（需手动创建）
├─ frontend/                 # React + Ant Design 前端
│  └─ src/
│     ├─ layouts/            # 应用布局
│     ├─ pages/              # 页面：登录、看板、缺陷、用户、日志、基础数据
│     ├─ context/            # 认证/报警上下文
│     └─ api.js              # 前端接口封装
├─ test_external.py          # 外部模型推送接口测试脚本
├─ start.sh                  # Linux / macOS 一键启动脚本
├─ start.bat                 # Windows 一键启动脚本
├─ docker-compose.yml        # Docker Compose 编排文件
├─ .env.example              # 后端环境变量示例
└─ README.md                 # 项目说明文档
```

## 一、环境准备

请先确认本机已安装以下环境：

- **Node.js 18+** 和 **npm 9+**：用于运行前端
- **Go 1.23+**：用于运行后端
- **Python 3.10+**：仅用于执行 `test_external.py` 测试脚本
- **Docker Desktop / Docker Compose**：仅当你希望用容器方式启动时需要

检查命令：

```bash
node -v
npm -v
go version
python --version
docker --version
```

## 二、获取项目代码

```bash
git clone <your-repo-url>
cd defect-detection-system
```

## 三、配置后端环境变量

后端运行前需要在 `backend/` 目录下准备 `.env` 文件。

### 方式 1：从示例文件复制

Linux / macOS：

```bash
cp .env.example backend/.env
```

Windows PowerShell：

```powershell
Copy-Item .env.example backend/.env
```

### 方式 2：手动创建 `backend/.env`

```env
EXTERNAL_API_KEY=your-api-key
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-email@example.com
SMTP_PASSWORD=your-email-password
SMTP_FROM=your-email@example.com
ALERT_EMAIL_TO=admin@example.com
ADMIN_USERNAME=admin
ADMIN_PASSWORD=Admin@123456
ADMIN_DISPLAY_NAME=系统管理员
```

字段说明：

- `EXTERNAL_API_KEY`：第三方模型调用外部推送接口时使用的 API Key
- `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` / `SMTP_FROM`：邮件服务器配置
- `ALERT_EMAIL_TO`：接收严重缺陷告警邮件的管理员邮箱
- `ADMIN_USERNAME` / `ADMIN_PASSWORD` / `ADMIN_DISPLAY_NAME`：首次初始化默认管理员账号

## 四、安装依赖

### 1. 安装后端 Go 依赖

```bash
cd backend
go mod tidy
cd ..
```

### 2. 安装前端 npm 依赖

```bash
cd frontend
npm install
cd ..
```

## 五、本地开发启动

建议分别启动后端和前端。

### 终端 1：启动后端

```bash
cd backend
go run .
```

后端默认监听：

- http://localhost:8000

### 终端 2：启动前端

```bash
cd frontend
npm start
```

前端默认访问地址：

- http://localhost:3000

### 默认演示账号

首次启动后，系统会自动确保以下演示账号存在：

- 管理员：`admin / Admin@123456`
- 质检员：`inspector1 / Inspector@123456`
- 查看员/工人查看账号：`viewer1 / Viewer@123456`

登录后可在右上角修改本人密码；管理员可维护用户、日志与基础数据。

### 默认演示基础数据

系统会自动确保以下主数据存在：

- 产线：`PL-A / 产线A`、`PL-B / 产线B`、`PL-C / 产线C`
- 设备：`EQ-A-01 / 冲压机1号`、`EQ-B-01 / 焊接机器人1号`、`EQ-C-01 / 喷涂机1号`
- 缺陷类型：`DT-CRACK / 裂纹`、`DT-WELD / 焊接异常`、`DT-SURFACE / 表面瑕疵`

### 默认演示缺陷

系统会自动确保以下示例缺陷存在，并已关联主数据 ID：

- `冲压面板边缘裂纹`
- `焊点虚焊偏移`
- `涂层气泡不均`

其中示例缺陷默认状态为“待分派”，因此处理人、验收人、指派时间、验收时间等流转字段在未操作前显示为 `-` 属于正常现象。

## 六、主要页面与权限

### 管理员

- 数据看板
- 缺陷列表 / 新增缺陷 / 缺陷详情
- 用户管理
- 操作日志
- 产线管理
- 设备管理
- 缺陷类型管理
- Excel/PDF 报表导出
- 缺陷指派 / 验收 / 关闭

### 质检员

- 数据看板
- 缺陷列表 / 新增缺陷 / 缺陷详情
- 产线 / 设备 / 缺陷类型查看
- 缺陷处理、提交验收
- 已分配缺陷相关流转操作
- Excel/PDF 报表导出

### 查看员

- 数据看板
- 缺陷列表 / 缺陷详情
- 产线 / 设备 / 缺陷类型查看
- 无新增、删除、用户管理、操作日志权限

## 七、缺陷业务流程

当前系统支持典型缺陷闭环流转：

1. 创建缺陷
2. 管理员指派处理人/验收人
3. 处理人开始处理并提交验收
4. 验收人验收通过或退回
5. 管理员关闭缺陷

缺陷详情页支持查看可执行动作与流转记录。

## 八、基础数据管理

系统提供以下基础数据模块：

- 产线管理：维护产线编码、名称、状态
- 设备管理：维护设备编码、名称、所属产线、型号、状态
- 缺陷类型管理：维护缺陷类型编码、名称、默认严重级别、状态

缺陷录入时可关联缺陷类型、产线、设备；后端会校验主数据是否存在且处于启用状态。

## 九、严重缺陷提醒

当前系统对严重缺陷提供多层提醒：

- 数据看板顶部持续红色告警条
- 顶部铃铛中展示未关闭严重缺陷
- 新建严重缺陷后立即触发前端弹窗提醒
- 浏览器桌面通知（需用户授权）
- 声音提示
- 邮件通知（需正确配置 SMTP）

如果浏览器未授权通知权限，则仍会显示页面内提醒和铃铛提醒。

## 十、操作日志 / 审计追踪

管理员可在“操作日志”页面查看审计记录。

当前已覆盖的关键事件包括：

- 登录成功 / 登录失败 / 退出登录
- 本人修改密码成功 / 失败
- 用户创建、更新、启用、禁用、重置密码
- 无权限访问被拒绝
- 外部上报 API Key 校验失败
- 部分缺陷相关关键操作

日志会记录：

- 操作时间
- 操作人 / 角色
- 模块 / 操作类型 / 结果
- 对象名称
- 摘要
- 请求路径 / 方法 / IP / User-Agent
- 详情 JSON（如字段变更）

## 十一、报表导出

### 1. 缺陷列表 Excel 导出

后端接口：

```http
GET /api/defects/export.xlsx
```

支持按列表筛选条件导出。

### 2. 缺陷详情 PDF 导出

后端接口：

```http
GET /api/defects/{id}/export.pdf
```

用于输出单条缺陷的详情报告。

## 十二、外部模型推送接口

后端启动后，在项目根目录执行：

```bash
python test_external.py
```

接口要求：

```http
POST http://localhost:8000/api/external/report
Content-Type: application/json
X-API-Key: your-api-key
```

请求体示例：

```json
{
  "image_base64": "data:image/png;base64,...",
  "defect_type": "裂纹",
  "severity": "严重",
  "location": "产线A-工位3",
  "confidence": 0.98,
  "description": "外部模型自动识别到疑似裂纹缺陷",
  "source": "第三方检测模型"
}
```

## 十三、运行自动化测试与构建

### 1. 后端测试

```bash
cd backend
go test ./...
```

### 2. 后端构建

```bash
cd backend
go build .
```

### 3. 前端构建

```bash
cd frontend
npm run build
```

> 前端构建时如果出现 `@antv` 相关 source map warning，通常是三方依赖告警，不影响构建产物生成。

## 十四、Docker 部署

### 1. 准备环境变量

先确保 `backend/.env` 已存在，可由 `.env.example` 复制得到。

### 2. 构建并启动容器

```bash
docker compose up --build
```

启动成功后访问：

- 前端：http://localhost:3000
- 后端：http://localhost:8000

### 3. 停止容器

```bash
docker compose down
```

## 十五、建议的 Git 工作流

为避免把运行产物和本地环境文件带进版本库，建议：

```bash
git pull
git checkout -b feature/your-change
```

开发完成后先检查：

```bash
git status
git diff
```

确认不要提交以下内容：

- `backend/defects.db`
- `backend/uploads/`
- `.claude/`
- `__pycache__/`
- 前端 `build/`
- `node_modules/`
- 本地 `.env`

提交前建议执行：

```bash
cd backend && go test ./...
cd ../frontend && npm run build
```

## 十六、常见问题

### 1. 前端能打开，但接口请求失败

请确认：

- 后端 `go run .` 已正常启动
- 后端监听的是 `http://localhost:8000`
- 浏览器 Network 里请求的接口地址是否正确

### 2. 首页严重缺陷提醒没有出现

请检查：

- 数据库中是否存在 `severity = critical` 且未关闭的缺陷
- 当前前端连接的是否是最新启动的后端服务
- 浏览器是否拦截了通知权限（桌面通知仅在授权后生效）
- 至少刷新一次页面以重新加载看板与铃铛数据

### 3. 外部推送接口返回 401

请检查：

- `backend/.env` 中是否配置了 `EXTERNAL_API_KEY`
- 请求 Header 里的 `X-API-Key` 是否和 `.env` 中一致

### 4. 图片上传后不显示

请确认：

- `backend/uploads/` 目录存在且后端有写入权限
- 返回的图片路径是否以 `/uploads/` 开头

### 5. 设备无法关联到产线

请检查：

- 设备是否属于所选产线
- 产线和设备是否处于启用状态

### 6. 操作日志为空

请先执行登录、用户管理、缺陷处理或外部上报等关键操作，再刷新“操作日志”页面查看。

## 十七、说明

- 当前后端为 **纯 Go 实现**
- Python 仅用于执行 `test_external.py` 接口测试脚本
- SQLite 数据库文件默认位于 `backend/defects.db`
- 上传图片默认保存在 `backend/uploads/`
- README 中截图占位路径建议放在 `docs/screenshots/`
