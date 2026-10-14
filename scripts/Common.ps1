$ErrorActionPreference = 'Stop'
$PortalRoot = Split-Path $PSScriptRoot -Parent
function Find-PortalTool([string]$Name, [string]$Portable) {
  $candidate = Join-Path $PortalRoot $Portable
  if (Test-Path -LiteralPath $candidate) { return $candidate }
  $command = Get-Command $Name -ErrorAction SilentlyContinue
  if ($command) { return $command.Source }
  throw "$Name is missing. See README.md for setup instructions."
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
