param([string]$Image = 'python:3.13-alpine')
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $projectRoot
try {
    $verificationRoot = Join-Path $projectRoot ('tmp/learning-' + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $verificationRoot -Force | Out-Null
    $cli = Join-Path $verificationRoot 'evotool.exe'
    $memory = Join-Path $verificationRoot 'memory'
    & go build -o $cli ./cmd/evotool
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }

    function Invoke-CLI([string[]]$CliArgs) {
        $output = & $cli @CliArgs --root $memory --image $Image
        if ($LASTEXITCODE -ne 0) { throw "CLI failed: $CliArgs" }
        return ($output | ConvertFrom-Json)
    }
    function Write-JSON($Value, [string]$File) {
        [IO.File]::WriteAllText($File, ($Value | ConvertTo-Json -Depth 30), (New-Object Text.UTF8Encoding($false)))
    }
    $learned = Invoke-CLI @('learn', '--package', 'examples/csv-analysis/generated.json')
    if (-not $learned.validation.Valid) { throw 'Validation did not pass' }
    $first = Invoke-CLI @('execute', '--id', 'csv_numeric_summary', '--input', 'examples/csv-analysis/input.json')
    $second = Invoke-CLI @('execute', '--id', 'csv_numeric_summary', '--input', 'examples/csv-analysis/input-second.json')
    $firstOutput = $first.Output | ConvertFrom-Json
    $secondOutput = $second.Output | ConvertFrom-Json
    if (-not $first.Success -or $firstOutput.total -ne 60 -or $firstOutput.average -ne 20) { throw 'First result mismatch' }
    if (-not $second.Success -or $secondOutput.total -ne 100 -or $secondOutput.average -ne 50) { throw 'Reuse result mismatch' }
    $search = & $cli search tools --query 'csv numeric summary' --root $memory
    if ($LASTEXITCODE -ne 0 -or ($search -join "`n") -notmatch 'selected: csv_numeric_summary') { throw 'Search did not select learned tool' }

    # New packages with failing tests or privileged permissions must not be published.
    foreach ($kind in @('test_failure', 'network_access')) {
        $candidate = Get-Content examples/csv-analysis/generated.json -Raw | ConvertFrom-Json
        $candidate.Manifest.Name = 'csv_' + $kind
        if ($kind -eq 'test_failure') { $candidate.Tests[0].Content = "raise AssertionError('intentional failure')" }
        else { $candidate.Manifest.Permissions.Network = 'full' }
        $packageFile = Join-Path $verificationRoot ($kind + '.json')
        Write-JSON $candidate $packageFile
        & $cli learn --package $packageFile --root $memory --image $Image
        if ($LASTEXITCODE -eq 0) { throw "Invalid package accepted: $kind" }
        if (Test-Path (Join-Path $memory ('tools/' + $candidate.Manifest.Name))) { throw "Rejected package persisted: $kind" }
    }
    & $cli learn --package examples/csv-analysis/generated.json --root $memory --image $Image
    if ($LASTEXITCODE -eq 0) { throw 'Duplicate submission overwrote existing tool' }
    $history = @(Get-Content (Join-Path $memory 'registry/execution_history/csv_numeric_summary.jsonl'))
    if ($history.Count -ne 2) { throw 'Expected two persisted execution records' }
    $metadata = Get-Content (Join-Path $memory 'registry/metadata/tools/csv_numeric_summary.json') -Raw | ConvertFrom-Json
    if (-not $metadata.Validation.Valid -or $metadata.CurrentVersion -ne 'v1') { throw 'Validation evidence missing from registry' }
    Write-Output 'PASS: validation, persistence, execution, search, reuse, history, failure rejection, permission rejection, overwrite protection'
    Write-Output "Evidence: $verificationRoot"
    exit 0
} finally { Pop-Location }
