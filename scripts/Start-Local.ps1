param([ValidateSet('dynamodb','file')][string]$Storage = 'dynamodb')
. "$PSScriptRoot/Common.ps1"
$processFile = Join-Path $PortalRoot '.run/processes.json'
if (Test-Path -LiteralPath $processFile) { throw 'Run scripts/Stop-Local.ps1 before starting again.' }
foreach ($port in @(8080,8082) + $(if ($Storage -eq 'dynamodb') { @(8000) } else { @() })) {
  $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback,$port)
  try { $listener.Start() } catch { throw "Port $port is busy. Stop the previous server before starting." } finally { $listener.Stop() }
}
$node = Find-PortalTool 'node.exe' '.tools/node-v24.21.0-win-x64/node.exe'
$env:PATH = (Split-Path $node -Parent) + ';' + $env:PATH
$npm = Join-Path (Split-Path $node -Parent) 'node_modules/npm/bin/npm-cli.js'
$go = (Get-Command go -ErrorAction Stop).Source
$run = Join-Path $PortalRoot '.run'
New-Item -ItemType Directory -Force "$run/bin", "$run/dynamodb" | Out-Null
Push-Location "$PortalRoot/frontend"
try {
  if (-not (Test-Path 'node_modules')) { Invoke-Checked $node @($npm,'ci','--cache',"$PortalRoot/.tools/npm-cache") }
  Invoke-Checked $node @($npm,'run','build')
} finally { Pop-Location }
Push-Location "$PortalRoot/backend/portal-api"
try {
  Invoke-Checked $go @('build','-o',"$run/bin/project-service.exe",'./cmd/project-service')
  Invoke-Checked $go @('build','-o',"$run/bin/portal-api.exe",'.')
} finally { Pop-Location }
$records = @()
function Start-PortalProcess([string]$Label, [string]$Executable, [string]$Arguments = '') {
  $options = @{ FilePath=$Executable; WorkingDirectory=$run; WindowStyle='Hidden'; PassThru=$true; RedirectStandardOutput="$run/$Label.out.log"; RedirectStandardError="$run/$Label.err.log" }
  if ($Arguments) { $options.ArgumentList=$Arguments }
  $process = Start-Process @options
  $script:records += @{id=$process.Id; executable=$Executable; started=$process.StartTime.ToUniversalTime().ToString('o')}
  ConvertTo-Json -InputObject @($script:records) | Set-Content -Encoding UTF8 -LiteralPath $processFile
}
$names = @('GIN_MODE','PROJECT_STORE','PROJECT_DATA_FILE','AWS_REGION','DYNAMODB_TABLE','DYNAMODB_ENDPOINT','DYNAMODB_CREATE_TABLE','PROJECT_LISTEN_ADDR','PORTAL_LISTEN_ADDR','PROJECT_SERVICE_URL','FRONTEND_DIR')
$savedEnv = @{}; foreach ($name in $names) { $savedEnv[$name] = [Environment]::GetEnvironmentVariable($name,'Process') }
try {
  $env:GIN_MODE='release'; $env:PROJECT_STORE=$Storage
  $env:PROJECT_LISTEN_ADDR='127.0.0.1:8082'; $env:PORTAL_LISTEN_ADDR='127.0.0.1:8080'
  $env:PROJECT_SERVICE_URL='http://127.0.0.1:8082'; $env:FRONTEND_DIR="$PortalRoot/frontend/dist"
  $env:PROJECT_DATA_FILE="$run/projects.json"
  if ($Storage -eq 'dynamodb') {
    $java = Get-ChildItem "$PortalRoot/.tools/java/*/bin/java.exe" -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
    if (-not $java -or -not (Test-Path "$PortalRoot/.tools/dynamodb/DynamoDBLocal.jar")) { throw 'Portable Java/DynamoDB tools are missing. Use -Storage file or follow README setup.' }
    Start-PortalProcess 'dynamodb' $java ('-Djava.library.path="' + "$PortalRoot/.tools/dynamodb/DynamoDBLocal_lib" + '" -jar "' + "$PortalRoot/.tools/dynamodb/DynamoDBLocal.jar" + '" -sharedDb -disableTelemetry -dbPath "' + "$run/dynamodb" + '" -port 8000')
    $env:AWS_REGION='ap-south-1';$env:DYNAMODB_TABLE='portal-projects';$env:DYNAMODB_ENDPOINT='http://127.0.0.1:8000';$env:DYNAMODB_CREATE_TABLE='true'
    $deadline = (Get-Date).AddSeconds(60)
    $connected = $false
    do {
      $socket = [Net.Sockets.TcpClient]::new()
      try { $socket.Connect('127.0.0.1',8000); $connected = $socket.Connected } catch {} finally { $socket.Dispose() }
      if (-not $connected) { Start-Sleep -Milliseconds 500 }
    } while (-not $connected -and (Get-Date) -lt $deadline)
    if (-not $connected) { throw 'DynamoDB Local did not start. See .run/dynamodb.err.log.' }
  }
  Start-PortalProcess 'project-service' "$run/bin/project-service.exe"
  Wait-PortalReady 'http://127.0.0.1:8082/readyz'
  Start-PortalProcess 'portal-api' "$run/bin/portal-api.exe"
  Wait-PortalReady 'http://127.0.0.1:8080/readyz'
  Write-Host "Portal is running: http://127.0.0.1:8080 (storage: $Storage)"
} catch {
  & "$PSScriptRoot/Stop-Local.ps1"
  throw
} finally { foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name,$savedEnv[$name],'Process') } }
