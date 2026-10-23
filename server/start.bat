@echo off
chcp 65001 >nul 2>&1
setlocal
title SecretProject Server - HTTPS 8090 / HTTP 8091
cd /d "%~dp0"

echo ==================================
echo   SecretProject - HTTPS 8090
echo   HTTP (local)   - 8091
echo ==================================
echo.

rem ---- 1. Kill anything holding our ports (and legacy 8080/8081 from old
rem builds), otherwise the new server cannot bind them.
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8090 " ^| findstr "LISTENING"') do taskkill /f /pid %%a >nul 2>&1
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8091 " ^| findstr "LISTENING"') do taskkill /f /pid %%a >nul 2>&1
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8080 " ^| findstr "LISTENING"') do taskkill /f /pid %%a >nul 2>&1
for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":8081 " ^| findstr "LISTENING"') do taskkill /f /pid %%a >nul 2>&1

rem ---- 2. MongoDB must be up: the server exits immediately (log.Fatal in
rem initDB) if 127.0.0.1:27017 is unreachable, so check before starting.
netstat -ano | findstr ":27017 " | findstr "LISTENING" >nul 2>&1
if errorlevel 1 (
    echo MongoDB не запущена. Пробую поднять службу MongoDB...
    net start MongoDB >nul 2>&1
    timeout /t 2 /nobreak >nul
    netstat -ano | findstr ":27017 " | findstr "LISTENING" >nul 2>&1
    if errorlevel 1 (
        echo.
        echo ОШИБКА: MongoDB недоступна на 127.0.0.1:27017.
        echo Запусти MongoDB вручную - служба "MongoDB" ^(services.msc^)
        echo или в консоли: mongod --dbpath путь_к_базе
        echo.
        pause
        exit /b 1
    )
)
echo MongoDB: OK

rem ---- 3. Rebuild from source so the latest version always runs.
where go >nul 2>&1
if %errorlevel%==0 (
    echo Building latest version from source...
    go build -o server.exe .
    if errorlevel 1 (
        echo Build FAILED - check the Go installation.
        pause
        exit /b 1
    )
)

rem ---- 4. Watchdog: if the server exits (crash, error), restart it.
:run
echo.
echo [%date% %time%] Запуск сервера... Закрой это окно или нажми Ctrl+C, чтобы остановить.
echo.
server.exe
echo.
echo [%date% %time%] Сервер остановился (код %errorlevel%). Перезапуск через 3 сек...
timeout /t 3 /nobreak >nul
goto run
