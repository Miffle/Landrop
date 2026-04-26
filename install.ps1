# Landrop Windows Installer (PowerShell)
# Usage: iex (irm https://raw.githubusercontent.com/Miffle/Landrop/main/install.ps1)

$ErrorActionPreference = "Stop"

$REPO    = "Miffle/Landrop"
$BIN     = "landrop.exe"
$API_URL = "https://api.github.com/repos/$REPO/releases/latest"

function Write-Info  { Write-Host "[landrop] $args" -ForegroundColor Cyan }
function Write-Ok    { Write-Host "[landrop] $args" -ForegroundColor Green }
function Write-Err   { Write-Host "[landrop] ERROR: $args" -ForegroundColor Red; exit 1 }

# Default install dir: %LOCALAPPDATA%\Landrop
$INSTALL_DIR = if ($env:LANDROP_DIR) { $env:LANDROP_DIR } else { "$env:LOCALAPPDATA\Landrop" }

Write-Info "Fetching latest release from GitHub..."
try {
    $release = Invoke-RestMethod -Uri $API_URL -Headers @{ "User-Agent" = "landrop-installer" }
} catch {
    Write-Err "Failed to fetch release info: $_"
}

$tag = $release.tag_name
Write-Info "Latest release: $tag"

$asset = $release.assets | Where-Object { $_.name -eq "landrop-windows-amd64.exe" } | Select-Object -First 1
if (-not $asset) { Write-Err "No Windows binary found in release $tag" }

$downloadUrl = $asset.browser_download_url

# Create install dir
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null

$dest = Join-Path $INSTALL_DIR $BIN
Write-Info "Downloading $($asset.name) to $dest..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $dest

Write-Ok "Installed: $dest"

# Add to user PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$INSTALL_DIR*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$INSTALL_DIR", "User")
    Write-Info "Added $INSTALL_DIR to user PATH (restart terminal to apply)"
}

Write-Ok "Done! Run: landrop"
Write-Info "Then open http://localhost:6437 in your browser."
