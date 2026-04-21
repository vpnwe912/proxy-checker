$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

$srcPng = Join-Path $root "assets-src\appicon.png"
$buildDir = Join-Path $root "build"
$buildWindowsDir = Join-Path $root "build\windows"
$dstPng = Join-Path $buildDir "appicon.png"
$dstIco = Join-Path $buildWindowsDir "icon.ico"

if (!(Test-Path $srcPng)) {
    throw "Source icon not found: $srcPng"
}

New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
New-Item -ItemType Directory -Force -Path $buildWindowsDir | Out-Null

Write-Host "Copying PNG to build/appicon.png"
Copy-Item $srcPng $dstPng -Force

Write-Host "Generating ICO to build/windows/icon.ico"
magick $srcPng -background none -define icon:auto-resize=16,24,32,40,48,64,72,96,128,256 $dstIco

Write-Host "Checking ICO sizes"
magick identify $dstIco

Write-Host "Assets prepared successfully"