.PHONY: prepare build rebuild clean dev check

prepare:
	powershell -ExecutionPolicy Bypass -File .\scripts\prepare-assets.ps1

build: prepare
	powershell -ExecutionPolicy Bypass -File .\scripts\auto-increment-version.ps1
	wails build -clean

rebuild: clean build

clean:
	if exist build\bin rmdir /s /q build\bin

dev: prepare
	wails dev

check:
	if not exist assets-src\appicon.png (echo ERROR: assets-src\appicon.png not found && exit /b 1)
	if not exist scripts\prepare-assets.ps1 (echo ERROR: scripts\prepare-assets.ps1 not found && exit /b 1)