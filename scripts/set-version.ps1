# Set version in wails.json
param(
    [Parameter(Mandatory=$true)]
    [string]$Version
)

$ErrorActionPreference = "Stop"

# Validate version format
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    Write-Host "Invalid version format. Use: X.Y.Z (e.g., 1.0.0)" -ForegroundColor Red
    exit 1
}

# Read wails.json
$wails = Get-Content "wails.json" -Raw | ConvertFrom-Json

# Update version
$wails.info.productVersion = $Version

# Save without BOM
$utf8NoBom = New-Object System.Text.UTF8Encoding $false
$json = $wails | ConvertTo-Json -Depth 10
[System.IO.File]::WriteAllText("$PWD\wails.json", $json, $utf8NoBom)

Write-Host "Version updated: $Version" -ForegroundColor Green
Write-Host "File: wails.json" -ForegroundColor Cyan
