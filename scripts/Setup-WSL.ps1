# Run from an administrator PowerShell window. Windows is never restarted here.
$ErrorActionPreference = 'Stop'
$portalRoot = Split-Path $PSScriptRoot -Parent
$runDirectory = Join-Path $portalRoot '.run'
New-Item -ItemType Directory -Force $runDirectory | Out-Null
$resultPath = Join-Path $runDirectory 'wsl-setup-result.json'
Start-Transcript -Path (Join-Path $runDirectory 'wsl-setup.log') -Force | Out-Null
try {
  $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = [Security.Principal.WindowsPrincipal]::new($identity)
  if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Open PowerShell using Run as administrator before running this script.'
  }
  # Microsoft's WSL 2 prerequisite. /norestart leaves reboot timing to the user.
  & dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart
  if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 3010) { throw "Virtual Machine Platform setup failed: $LASTEXITCODE" }
  $offlineInstaller = Join-Path $portalRoot '.tools/wsl.msi'
  if (Test-Path -LiteralPath $offlineInstaller) {
    $expectedHash = 'A3505A50F4CC585551D11D9DE824BA4375448D7A68F2E71D3FB315FA986FC754'
    if ((Get-FileHash -LiteralPath $offlineInstaller -Algorithm SHA256).Hash -ne $expectedHash) { throw 'WSL offline installer checksum mismatch.' }
    $signature = Get-AuthenticodeSignature -LiteralPath $offlineInstaller
    if ($signature.Status -ne 'Valid' -or $signature.SignerCertificate.Subject -notmatch 'Microsoft') { throw 'WSL offline installer signature is not valid.' }
    $installLog = Join-Path $runDirectory 'wsl-msi-install.log'
    $installer = Start-Process -FilePath 'msiexec.exe' -ArgumentList @('/i', ('"' + $offlineInstaller + '"'), '/qn', '/norestart', '/l*v', ('"' + $installLog + '"')) -WindowStyle Hidden -Wait -PassThru
    if ($installer.ExitCode -ne 0 -and $installer.ExitCode -ne 3010) { throw "WSL offline installation failed: $($installer.ExitCode)" }
  } else {
    & winget.exe install --source winget --id Microsoft.WSL --exact --installer-type wix --silent --accept-package-agreements --accept-source-agreements --disable-interactivity
    if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 3010) { throw "WSL package installation failed: $LASTEXITCODE" }
  }
  @{status='installed'; restartRequired=$true; message='Save your work and restart Windows yourself. Then open Docker Desktop.'} | ConvertTo-Json | Set-Content -Encoding UTF8 -LiteralPath $resultPath
  Write-Host 'WSL setup finished. Save your work and restart Windows yourself, then open Docker Desktop.'
} catch {
  @{status='failed'; message=$_.Exception.Message} | ConvertTo-Json | Set-Content -Encoding UTF8 -LiteralPath $resultPath
  throw
} finally { Stop-Transcript | Out-Null }
