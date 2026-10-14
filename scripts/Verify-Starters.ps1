. "$PSScriptRoot/Common.ps1"
$go = (Get-Command go -ErrorAction Stop).Source
$node = Find-PortalTool 'node.exe' '.tools/node-v24.21.0-win-x64/node.exe'
$env:PATH=(Split-Path $node -Parent)+';'+$env:PATH
$npm=Join-Path (Split-Path $node -Parent) 'node_modules/npm/bin/npm-cli.js'
$checkRoot=Join-Path $PortalRoot ('.run/starter-check-'+(Get-Date).ToString('yyyyMMddHHmmss'))
New-Item -ItemType Directory -Force $checkRoot | Out-Null
foreach ($kind in @('api','ui')) {
  $zip=Join-Path $PortalRoot ".run/browser/$kind.zip"
  if (-not(Test-Path -LiteralPath $zip)) {throw 'Run the frontend browser tests first to produce starter ZIPs.'}
  Expand-Archive -LiteralPath $zip -DestinationPath "$checkRoot/$kind"
}
Push-Location "$checkRoot/api"
try {
  Invoke-Checked $go @('mod','tidy')
  Invoke-Checked $go @('test','./...')
  Invoke-Checked $go @('build','-o',"$checkRoot/generated-api.exe",'.')
} finally {Pop-Location}
$listener=[Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,8081)
try {$listener.Start()} catch {throw 'Port 8081 is busy; cannot verify the generated Go API.'} finally {$listener.Stop()}
$process=Start-Process -FilePath "$checkRoot/generated-api.exe" -WorkingDirectory "$checkRoot/api" -WindowStyle Hidden -PassThru -RedirectStandardOutput "$checkRoot/go.out.log" -RedirectStandardError "$checkRoot/go.err.log"
try {
  Wait-PortalReady 'http://127.0.0.1:8081/healthz' 20
  $health=Invoke-RestMethod 'http://127.0.0.1:8081/healthz'
  if($health.status -ne 'ok'){throw 'Generated API health check failed.'}
} finally {if(-not $process.HasExited){Stop-Process -Id $process.Id}}
Push-Location "$checkRoot/ui"
try {
  Invoke-Checked $node @($npm,'install','--cache',"$PortalRoot/.tools/npm-cache")
  Invoke-Checked $node @($npm,'run','build')
  Invoke-Checked $node @($npm,'audit','--audit-level=high')
} finally {Pop-Location}
Write-Host "PASS: downloaded Go API builds and responds to /healthz; downloaded React app builds. Artifacts: $checkRoot"
