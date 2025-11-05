@echo off
echo ===================================
echo  Step 1: Generating API docs...
echo ===================================
C:\Users\86184\go\bin\swag.exe init

echo.
echo.
echo ===================================
echo  Step 3: Starting Go server...
echo ===================================
go run main.go