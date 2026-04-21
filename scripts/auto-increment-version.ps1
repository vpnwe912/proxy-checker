# Automatic version increment
$ErrorActionPreference = "Stop"

# Read wails.json
$wails = Get-Content "wails.json" -Raw | ConvertFrom-Json

# Parse current version
$currentVersion = $wails.info.productVersion
$parts = $currentVersion -split '\.'
$major = [int]$parts[0]
$minor = [int]$parts[1]
$patch = [int]$parts[2]

Write-Host "Current version: $currentVersion" -ForegroundColor Cyan

# Increase patch
$patch++

# If patch reaches 10, increase minor and reset patch
if ($patch -ge 10) {
    $minor++
    $patch = 0
    Write-Host "Patch reached 10, increasing minor" -ForegroundColor Yellow
}

# Form new version
$newVersion = "$major.$minor.$patch"

# Update version in wails.json
$wails.info.productVersion = $newVersion

# Save without BOM
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
$json = $wails | ConvertTo-Json -Depth 10
[System.IO.File]::WriteAllText("$PWD\wails.json", $json, $utf8NoBom)

Write-Host "New version: $newVersion" -ForegroundColor Green
