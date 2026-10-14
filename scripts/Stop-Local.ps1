. "$PSScriptRoot/Common.ps1"
$processFile = Join-Path $PortalRoot '.run/processes.json'
if (-not (Test-Path -LiteralPath $processFile)) { Write-Host 'No recorded portal processes.'; return }
$records = @(Get-Content -LiteralPath $processFile -Raw | ConvertFrom-Json)
foreach ($record in $records) {
  $process = Get-Process -Id $record.id -ErrorAction SilentlyContinue
  if ($process -and $process.Path -eq $record.executable -and $process.StartTime.ToUniversalTime().ToString('o') -eq $record.started) {
    Stop-Process -Id $process.Id
    $process.WaitForExit(10000) | Out-Null
    Write-Host "Stopped $($record.executable)"
  }
}
Remove-Item -LiteralPath $processFile
Write-Host 'Project data has been kept.'
