# 缺陷检测管理系统

一个基于 **React + Ant Design + Go + SQLite** 的缺陷检测管理系统，支持缺陷录入、列表管理、数据看板、严重缺陷报警、外部模型推送、邮件通知、用户登录、角色权限控制和用户管理。

## 功能特性

- 数据看板：缺陷总数、等级统计、类型分布、近 7 天趋势
- 缺陷管理：新增缺陷、列表筛选、详情查看、状态更新、删除记录
- 报警通知：严重缺陷全屏弹窗、顶部铃铛提醒、浏览器桌面通知、声音提醒
- 外部推送：第三方检测模型通过 `POST /api/external/report` 自动上报缺陷
- 邮件通知：创建严重缺陷后自动向管理员邮箱发送告警邮件
- 用户登录：账号密码登录、退出登录、本人修改密码
- 角色权限：管理员、质检员、查看员 3 类角色访问控制
- 用户管理：管理员可新增用户、修改角色、启用/禁用账号、重置密码

## 技术栈

- 前端：React 19、Ant Design 5、Ant Design Charts、Axios
- 后端：Go 1.23、net/http、modernc.org/sqlite
- 数据库：SQLite
- 辅助脚本：Python 3（仅用于运行 `test_external.py`）
- 部署：Docker / Docker Compose

## 项目结构

```text
.
├─ backend/              # Go 后端服务
│  ├─ main.go            # 后端主程序
│  ├─ main_test.go       # Go 自动化测试
│  ├─ go.mod             # Go 依赖配置
│  ├─ defects.db         # SQLite 数据库文件（运行后生成/更新）
│  ├─ uploads/           # 上传图片目录（运行后生成/写入）
│  ├─ Dockerfile         # 后端镜像构建文件
│  └─ .env               # 后端环境变量（需手动创建）
├─ frontend/             # React + Ant Design 前端
├─ test_external.py      # 外部模型推送接口测试脚本
├─ start.sh              # Linux / macOS 一键启动脚本
├─ start.bat             # Windows 一键启动脚本
├─ docker-compose.yml    # Docker Compose 编排文件
├─ .env.example          # 后端环境变量示例
└─ README.md             # 项目说明文档
```

## 一、环境准备

请先确认本机已安装以下环境：

- **Node.js 18+** 和 **npm 9+**：用于运行前端
- **Go 1.22+ / 1.23+**：用于运行后端
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

如果你已经在当前项目目录中，可以直接进入下一步。

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

参考下面内容填写：

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

如果暂时不需要邮件通知，可以先保留占位值；后端会在邮件配置不完整时跳过邮件发送，但外部推送接口仍建议配置 `EXTERNAL_API_KEY`。

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

需要分别启动后端和前端，建议开两个终端窗口。

### 终端 1：启动后端

```bash
cd backend
go run .
```

后端默认监听：

- http://localhost:8000

可以用浏览器打开 `http://localhost:8000/`，如果返回 API 信息，说明后端启动成功。

### 终端 2：启动前端

```bash
cd frontend
npm start
```

前端默认访问地址：

- http://localhost:3000

浏览器打开后会先进入登录页。

### 默认演示账号

首次启动时，如果数据库中还没有用户，系统会自动初始化默认管理员：

- 用户名：`admin`
- 密码：`Admin@123456`

登录后可以在右上角修改本人密码；管理员还可以进入“用户管理”页面创建质检员和查看员账号。

## 六、一键启动脚本

### Windows

在项目根目录执行：

```bat
.\start.bat
```

脚本会分别打开两个命令行窗口，自动执行：

- 后端：`go run .`
- 前端：`npm start`

### Linux / macOS

在项目根目录执行：

```bash
bash start.sh
```

## 七、运行自动化测试

### 1. 运行 Go 后端测试

```bash
cd backend
go test ./...
```

### 2. 构建 Go 后端

```bash
cd backend
go build .
```

### 3. 构建前端生产包

```bash
cd frontend
npm run build
```

## 八、验证核心功能

### 1. 验证登录与权限

打开 http://localhost:3000 后，依次检查：

- 未登录是否自动跳转到登录页
- 使用 `admin / Admin@123456` 是否可以正常登录
- 登录后右上角是否显示当前用户、角色、退出按钮、修改密码按钮
- 管理员是否可进入“用户管理”页面
- 管理员是否可新增质检员、查看员账号
- 质检员是否可新增/编辑缺陷但不能删除
- 查看员是否只能查看看板、缺陷列表和详情

### 2. 验证前端页面

登录后依次检查：

- 数据看板是否正常展示统计卡片和图表
- 缺陷列表是否能正常加载、筛选、查看详情
- 管理员/质检员是否可新增缺陷
- 管理员是否可删除缺陷
- 查看员是否看不到新增、删除等操作入口

### 3. 验证本人修改密码

登录系统后点击右上角“修改密码”，检查：

- 输入正确原密码后能否修改成功
- 新密码能否重新登录
- 原密码是否失效

### 4. 验证严重缺陷报警

新增一条严重等级为 **严重 / critical** 的缺陷后，检查：

- 页面是否弹出严重缺陷报警弹窗
- 顶部铃铛是否出现未处理严重缺陷数量
- 浏览器是否有桌面通知提示（需要先允许通知权限）
- 是否有提示音播放
- 若邮件 SMTP 配置正确，管理员邮箱是否收到告警邮件

### 5. 验证外部模型推送接口

后端启动后，在项目根目录执行：

```bash
python test_external.py
```

该脚本会向后端接口 `POST /api/external/report` 发送一条模拟缺陷数据。

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

成功后，前端缺陷列表中会新增一条来自外部模型的缺陷记录；如果等级为严重，还会触发报警通知。

## 九、Docker 部署

### 1. 准备环境变量

先确保 `backend/.env` 已存在，可由 `.env.example` 复制得到。

### 2. 构建并启动容器

在项目根目录执行：

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

## 十、常见问题

### 1. 端口 3000 或 8000 被占用

请先关闭占用端口的进程，或者修改前后端启动端口配置后再启动。

### 2. Windows 执行 `start.bat` 提示找不到命令

如果你在 PowerShell 中运行，请使用：

```powershell
.\start.bat
```

不要直接输入 `start.bat`。

### 3. 前端能打开，但接口请求失败

请确认：

- 后端 `go run .` 已正常启动
- 后端监听的是 `http://localhost:8000`
- 浏览器 Network 里请求的接口地址是否正确

### 4. 外部推送接口返回 401

请检查：

- `backend/.env` 中是否配置了 `EXTERNAL_API_KEY`
- 请求 Header 里的 `X-API-Key` 是否和 `.env` 中一致

### 5. 严重缺陷没有发送邮件

请检查 `backend/.env` 中 SMTP 配置是否完整、账号密码/授权码是否正确、邮箱服务商是否允许 SMTP 登录。

### 6. 图片上传后不显示

请确认 `backend/uploads/` 目录存在且后端有写入权限，并检查返回的图片路径是否以 `/uploads/` 开头。

## 十一、说明

- 当前后端已切换为 **纯 Go 实现**，不再依赖 Python/Conda 作为后端运行环境
- Python 仅用于执行 `test_external.py` 接口测试脚本
- SQLite 数据库文件为 `backend/defects.db`
- 上传图片保存在 `backend/uploads/`
- README 中截图占位路径建议放在 `docs/screenshots/`

## 截图占位符

![数据看板截图占位](docs/screenshots/dashboard.png)
![缺陷列表截图占位](docs/screenshots/defects.png)
![新增缺陷截图占位](docs/screenshots/new-defect.png)
![缺陷详情截图占位](docs/screenshots/defect-detail.png)
