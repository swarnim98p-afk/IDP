. "$PSScriptRoot/Common.ps1"
$docker = Find-PortalTool 'docker.exe' '.tools/docker/docker.exe'
$node = Find-PortalTool 'node.exe' '.tools/node-v24.21.0-win-x64/node.exe'
$engineType = & $docker info --format '{{.OSType}}'
if ($LASTEXITCODE -ne 0) { throw 'Open Docker Desktop and wait for its engine to start before running container checks.' }
if ($engineType -ne 'linux') { throw 'Switch Docker Desktop to Linux containers.' }
Push-Location $PortalRoot
try {
  Invoke-Checked $docker @('compose','config','--quiet')
  Invoke-Checked $docker @('compose','up','--build','-d','--wait','--wait-timeout','180')
  Wait-PortalReady 'http://127.0.0.1:8083/readyz'
} finally { Pop-Location }
$savedURL = $env:PORTAL_TEST_URL
$savedBrowserPath = $env:PLAYWRIGHT_BROWSERS_PATH
try {
  $env:PORTAL_TEST_URL = 'http://127.0.0.1:8083'
  $env:PLAYWRIGHT_BROWSERS_PATH = Join-Path $PortalRoot '.tools/playwright-browsers'
  Push-Location "$PortalRoot/frontend"
  try { Invoke-Checked $node @('tests/e2e.mjs') } finally { Pop-Location }
  & "$PSScriptRoot/Verify-Starters.ps1"
  Write-Host 'PASS: Linux containers, readiness, browser flows, and downloaded starter builds.'
  Write-Host 'Container portal is available at http://127.0.0.1:8083'
} finally {
  $env:PORTAL_TEST_URL = $savedURL
  $env:PLAYWRIGHT_BROWSERS_PATH = $savedBrowserPath
}
