# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Industrial defect detection and management system (中文界面). Go backend + React frontend. Tracks defects through a full workflow lifecycle: creation → assignment → processing → acceptance → closure.

## Commands

### Backend (Go)
```bash
cd backend && go run .          # Run dev server (port 8000)
cd backend && go build .        # Build binary
cd backend && go test ./...     # Run all tests
cd backend && go test -run TestFunctionName ./...  # Run single test
cd backend && go mod tidy       # Sync dependencies
```

### Frontend (React)
```bash
cd frontend && npm install      # Install dependencies
cd frontend && npm start        # Dev server (port 3000)
cd frontend && npm run build    # Production build
cd frontend && npm test         # Run tests (Jest)
```

### Docker
```bash
docker compose up --build       # Run full stack (frontend:3000, backend:8000)
```

### Quick Start
```bash
# Windows
start.bat
# Linux/macOS
./start.sh
```

## Architecture

### Backend (`backend/`)
- **Single-package Go monolith** — all code in `package main`, no framework (standard `net/http`)
- **Two main source files**: `main.go` (core: auth, defects, users, exports, ~2800 lines) and `masterdata.go` (production lines, equipment, defect types CRUD, ~720 lines)
- **Database**: SQLite via `modernc.org/sqlite` (pure Go, no CGo). File: `backend/defects.db`. Auto-creates schema and seeds demo data on startup.
- **8 tables**: `defects`, `users`, `sessions`, `production_lines`, `equipment`, `defect_types`, `operation_logs`, `defect_flow_logs`
- **Auth**: Bearer token in `Authorization` header. Tokens are random hex, SHA-256 hashed for storage. Middleware chain: `withCORS` → `withAuth` → `requireRoles` → handler
- **Routing pattern**: Handlers check `r.Method` internally (GET/POST/PUT/DELETE switch), not separate registrations
- **Tests**: `main_test.go` uses `httptest` with temp in-memory SQLite databases

### Frontend (`frontend/`)
- **React 19 + Ant Design 6** (Create React App, react-scripts 5)
- **Routing**: react-router-dom v7, routes defined in `App.js`
- **State**: React Context API only — `AuthContext` (token in localStorage key `defect-system-token`) and `AlertContext` (polls `/api/alerts` every 30s, triggers sound/desktop notifications)
- **API layer**: `src/api.js` — Axios instance pointed at `http://localhost:8000`, auto-attaches Bearer token, auto-clears on 401
- **Reusable CRUD component**: `MasterDataManagePage` powers production lines, equipment, and defect type management pages
- **Charts**: `@ant-design/charts` for dashboard visualizations (pie chart, trend line)

### Communication
Frontend calls backend REST API directly (no proxy). CORS handled server-side. All API routes under `/api/`.

### Defect Workflow State Machine
```
open → (assign) → assigned → (start_process) → processing → (submit_for_acceptance) → pending_acceptance → (accept) → accepted → (close) → closed
                                                              pending_acceptance → (reject) → rejected → (reassign) → assigned
```
Transitions enforced by `defectActionTransitions` map. Each transition logged to `defect_flow_logs`.

### Roles
- `admin` — full access including user management and operation logs
- `inspector` — can create/manage defects
- `viewer` — read-only access

## Environment Variables

Backend reads from `backend/.env` (see `.env.example`):
- `EXTERNAL_API_KEY` — API key for external report endpoint (`/api/external/report`)
- `SMTP_*` — Mail server config for critical defect email alerts
- `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `ADMIN_DISPLAY_NAME` — initial admin account

## Key Patterns

- All user-facing text is in Chinese (中文)
- Direct SQL queries (no ORM) — write raw SQL for database operations
- PDF export uses `C:/Windows/Fonts/simhei.ttf` for Chinese rendering (Windows-specific path)
- File uploads (defect images) stored in `backend/uploads/`, served via `http.FileServer`
- Operation logs provide full audit trail for security-sensitive actions
