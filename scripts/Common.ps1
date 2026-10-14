$ErrorActionPreference = 'Stop'
$PortalRoot = Split-Path $PSScriptRoot -Parent
function Find-PortalTool([string]$Name, [string]$Portable) {
  $candidate = Join-Path $PortalRoot $Portable
  $resolvedTool = $null
  if (Test-Path -LiteralPath $candidate) { $resolvedTool = $candidate }
  else {
    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($command) { $resolvedTool = $command.Source }
    elseif ($Name -eq 'docker.exe' -and (Test-Path 'C:/Program Files/Docker/Docker/resources/bin/docker.exe')) { $resolvedTool = 'C:/Program Files/Docker/Docker/resources/bin/docker.exe' }
  }
  if (-not $resolvedTool) { throw "$Name is missing. See README.md for setup instructions." }
  if ($Name -eq 'docker.exe') {
    # Docker invokes its credential helper by name, including from older terminals.
    $env:PATH = (Split-Path $resolvedTool -Parent) + ';' + $env:PATH
  }
  return $resolvedTool
}
function Invoke-Checked([string]$Executable, [string[]]$Arguments) {
  & $Executable @Arguments
  if ($LASTEXITCODE -ne 0) { throw "$Executable failed with exit code $LASTEXITCODE" }
}
function Wait-PortalReady([string]$URL, [int]$Seconds = 45) {
  $deadline = (Get-Date).AddSeconds($Seconds)
  do {
    try { $response = Invoke-WebRequest -UseBasicParsing $URL -TimeoutSec 2; if ($response.StatusCode -eq 200) { return } } catch {}
    Start-Sleep -Milliseconds 500
  } while ((Get-Date) -lt $deadline)
  throw "Service did not become ready at $URL. Check .run/*.err.log."
}
