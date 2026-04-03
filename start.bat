@echo off
setlocal

set "PROJECT_ROOT=%~dp0"

echo Starting backend: http://localhost:8000
start "defect-backend" /D "%PROJECT_ROOT%backend" cmd /k go run .

echo Starting frontend: http://localhost:3000
start "defect-frontend" /D "%PROJECT_ROOT%frontend" cmd /k npm start

echo.
echo Launch commands sent.
echo Backend: http://localhost:8000
echo Frontend: http://localhost:3000
echo.
pause
endlocal
