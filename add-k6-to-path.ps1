# PowerShell script to add k6 to PATH
# Run this script as Administrator

# Set the k6 installation path (change this to where you extracted k6)
$k6Path = "C:\k6"

# Check if path exists
if (-Not (Test-Path "$k6Path\k6.exe")) {
    Write-Host "Error: k6.exe not found at $k6Path\k6.exe" -ForegroundColor Red
    Write-Host "Please extract k6 to this folder or update `$k6Path in this script" -ForegroundColor Yellow
    exit 1
}

# Get current user's PATH
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")

# Check if already in PATH
if ($currentPath -like "*$k6Path*") {
    Write-Host "k6 is already in your PATH" -ForegroundColor Yellow
    exit 0
}

# Add to PATH
$newPath = $currentPath + ";$k6Path"
[Environment]::SetEnvironmentVariable("Path", $newPath, "User")

Write-Host "k6 has been added to your PATH" -ForegroundColor Green
Write-Host "Please close and reopen your terminal for changes to take effect" -ForegroundColor Yellow

