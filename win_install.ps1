$ProjectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ServerBin = Join-Path $ProjectDir "linux\server.exe"
$CliBin = Join-Path $ProjectDir "linux\cli.exe"

Write-Host "[*] Checking binaries..."
if (!(Test-Path $ServerBin)) { Write-Error "server.exe not found"; exit 1 }
if (!(Test-Path $CliBin)) { Write-Error "cli.exe not found"; exit 1 }

Write-Host "[*] Creating scheduled task for server..."

$Action = New-ScheduledTaskAction `
    -Execute $ServerBin

$Trigger = New-ScheduledTaskTrigger `
    -AtLogOn

$Principal = New-ScheduledTaskPrincipal `
    -UserId $env:USERNAME `
    -LogonType Interactive `
    -RunLevel LeastPrivilege

Register-ScheduledTask `
    -TaskName "ipmsg-server" `
    -Action $Action `
    -Trigger $Trigger `
    -Principal $Principal `
    -Force

Write-Host "[*] Starting server..."
Start-Process $ServerBin

Write-Host "[*] Adding cli to PATH as ipmsg..."

$TargetDir = "C:\Program Files\ipmsg"
New-Item -ItemType Directory -Force -Path $TargetDir | Out-Null
Copy-Item $CliBin "$TargetDir\ipmsg.exe" -Force

$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
if ($CurrentPath -notlike "*$TargetDir*") {
    [Environment]::SetEnvironmentVariable(
        "Path",
        "$CurrentPath;$TargetDir",
        "Machine"
    )
}

Write-Host "✅ Installation completed successfully!"
Write-Host "  • server runs in background and starts on login"
Write-Host "  • cli available globally as: ipmsg"
Write-Host "⚠️ Restart terminal to apply PATH changes"
