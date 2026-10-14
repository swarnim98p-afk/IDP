. "$PSScriptRoot/Common.ps1"
$processFile = Join-Path $PortalRoot '.run/processes.json'
if (-not (Test-Path -LiteralPath $processFile)) { Write-Host 'No recorded portal processes.'; return }
$records = Get-Content -LiteralPath $processFile -Raw | ConvertFrom-Json
foreach ($record in $records) {
  $process = Get-Process -Id $record.id -ErrorAction SilentlyContinue
  if ($process -and $process.Path -and [IO.Path]::GetFullPath($process.Path) -eq [IO.Path]::GetFullPath($record.executable) -and [Math]::Abs(($process.StartTime.ToUniversalTime() - [DateTime]::Parse($record.started).ToUniversalTime()).TotalSeconds) -lt 1) {
    Stop-Process -Id $process.Id
    $process.WaitForExit(10000) | Out-Null
    Write-Host "Stopped $($record.executable)"
  } elseif ($process) {
    Write-Host "Skipped process $($record.id): its identity differs from the recorded portal process."
  }
}
Remove-Item -LiteralPath $processFile
Write-Host 'Project data has been kept.'
