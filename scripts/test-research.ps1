# Test the Excavate research pipeline end to end.
# Usage:  powershell -ExecutionPolicy Bypass -File scripts/test-research.ps1
# Auto-starts Postgres/Redis (docker) and the backend if they are not running.
# Exit code 0 = all tests passed, 1 = one or more failed.

$ErrorActionPreference = "Continue"   # native stderr is noisy; we check codes explicitly

function Fail($msg) {
    Write-Host "ERROR: $msg" -ForegroundColor Red
    exit 1
}

if (-not (Get-Command curl.exe -ErrorAction SilentlyContinue)) {
    Fail "curl.exe not found (Windows ships it at C:\Windows\System32\curl.exe)"
}

$Root = Split-Path -Parent $PSScriptRoot
$Backend = Join-Path $Root "backend"
$BinDir = Join-Path $Backend "bin"
$Base = "http://localhost:8080"
$Cookie = Join-Path $env:TEMP "excavate_research_cookies_$PID.txt"
$Tmp = Join-Path $env:TEMP "excavate_research"

# ---- 1. Databases -----------------------------------------------------------
Write-Host "== Starting Postgres + Redis ==" -ForegroundColor Cyan
docker compose up -d postgres redis 2>&1 | Out-Null
$ready = $false
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep 2
    $ps = docker compose ps --format "{{.Name}} {{.Status}}"
    if (($ps | Select-String "healthy").Count -ge 2) { $ready = $true; break }
}
if (-not $ready) { Fail "Postgres/Redis did not become healthy in 60s." }
Write-Host "Databases healthy." -ForegroundColor Green

# ---- 2. Backend (auto-managed) ---------------------------------------------
$startedServer = $false
if (Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue) {
    Write-Host "Backend already listening on :8080 - reusing it." -ForegroundColor Yellow
} else {
    Write-Host "Building backend..." -ForegroundColor Cyan
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    Push-Location $Backend
    go build -o (Join-Path $BinDir "server.exe") ./cmd/server
    if ($LASTEXITCODE -ne 0) { Pop-Location; Fail "backend build failed" }
    Pop-Location

    $proc = Start-Process -FilePath (Join-Path $BinDir "server.exe") -WorkingDirectory $Backend -PassThru -WindowStyle Hidden
    $startedServer = $true
    $up = $false
    for ($i = 0; $i -lt 20; $i++) {
        Start-Sleep 1
        try { $null = Invoke-WebRequest -UseBasicParsing "$Base/api/healthz" -TimeoutSec 2; $up = $true; break } catch {}
    }
    if (-not $up) { Fail "backend did not start on :8080" }
    Write-Host "Backend started (PID $($proc.Id))." -ForegroundColor Green
}

# ---- 3. HTTP helpers --------------------------------------------------------
function PostJson([string]$path, [string]$body) {
    $outFile = Join-Path $Tmp "resp.txt"
    $bodyFile = Join-Path $Tmp "body.json"
    New-Item -ItemType Directory -Path $Tmp -Force | Out-Null
    Set-Content -Path $bodyFile -Value $body -NoNewline -Encoding ascii
    $curlArgs = @("-s", "-b", $Cookie, "-o", $outFile, "-w", "%{http_code}",
        "-H", "Content-Type: application/json", "--data-binary", "@$bodyFile", "$Base$path")
    $code = & curl.exe @curlArgs
    $resp = ""; if (Test-Path $outFile) { $resp = Get-Content $outFile -Raw }
    return @{ code = [string]$code; body = $resp }
}

function GetPath([string]$path) {
    $outFile = Join-Path $Tmp "resp.txt"
    New-Item -ItemType Directory -Path $Tmp -Force | Out-Null
    $curlArgs = @("-s", "-b", $Cookie, "-o", $outFile, "-w", "%{http_code}", "$Base$path")
    $code = & curl.exe @curlArgs
    $resp = ""; if (Test-Path $outFile) { $resp = Get-Content $outFile -Raw }
    return @{ code = [string]$code; body = $resp }
}

$results = @()
function Check([string]$name, [bool]$ok, [string]$detail) {
    $script:results += [pscustomobject]@{ Name = $name; Pass = $ok; Detail = $detail }
}

# ---- 4. Run the research flow ----------------------------------------------
Write-Host "== Running research flow ==" -ForegroundColor Cyan
$email = "research-$PID@test.com"
$password = "password123"
$query = "What is Redis and how does caching work?"

# Register + login to get a cookie.
$body = "{`"email`":`"$email`",`"password`":`"$password`"}"
$loginBodyFile = Join-Path $Tmp "login.json"
New-Item -ItemType Directory -Path $Tmp -Force | Out-Null
Set-Content -Path $loginBodyFile -Value $body -NoNewline -Encoding ascii
& curl.exe -s -c $Cookie -o NUL -H "Content-Type: application/json" --data-binary "@$loginBodyFile" "$Base/api/auth/register"
& curl.exe -s -c $Cookie -o NUL -H "Content-Type: application/json" --data-binary "@$loginBodyFile" "$Base/api/auth/login"

# Create a thread.
$r = PostJson "/api/threads" "{`"title`":`"Research test`"}"
if ($r.code -ne "201") { Check "create thread" $false "status $($r.code): $($r.body)" }
else {
    Check "create thread" $true "201"
    $threadId = ($r.body | ConvertFrom-Json).thread.id

    # Register + login
    $r = PostJson "/api/messages" "{`"threadId`":`"$threadId`",`"content`":`"$query`"}"
    if ($r.code -ne "201") { Check "post message" $false "status $($r.code): $($r.body)" }
    else {
        Check "post message" $true "201"
        $messageId = ($r.body | ConvertFrom-Json).message.id

        # Open the SSE stream in a background job and capture whatever arrives.
        $streamFile = Join-Path $Tmp "stream.txt"
        $streamUrl = "$Base/api/research/stream?messageID=$messageId"
        $job = Start-Job -ScriptBlock {
            & curl.exe -s -N -b $using:Cookie --max-time 25 $using:StreamUrl -o $using:StreamFile
        }
        # Poll until the message reaches a terminal state (up to 20s).
        $term = "pending"
        for ($i = 0; $i -lt 40; $i++) {
            Start-Sleep -Milliseconds 500
            $m = GetPath "/api/threads/$threadId"
            if ($m.code -eq "200") {
                $last = ($m.body | ConvertFrom-Json).messages | Where-Object { $_.id -eq $messageId }
                if ($last) { $term = $last.status; if ($term -eq "complete" -or $term -eq "error") { break } }
            }
        }
        Start-Sleep 1
        Receive-Job $job -ErrorAction SilentlyContinue | Out-Null
        Remove-Job $job -Force -ErrorAction SilentlyContinue
        $stream = Get-Content $streamFile -Raw -ErrorAction SilentlyContinue

        Check "message reached final state" ($term -eq "complete") $term
        Check "stream contains done event" ($stream -match '"type":"done"') $term

        # Re-read after completion for the authoritative answer + sources.
        $r2 = GetPath "/api/threads/$threadId"
        $ownerMsgs2 = (($r2.body | ConvertFrom-Json).messages | Where-Object { $_.id -eq $messageId })
        if ($ownerMsgs2) {
            Check "persisted answer non-empty" (($ownerMsgs2.content.Length -gt 200)) "len=$($ownerMsgs2.content.Length)"
            Check "persisted sources == 4" (($ownerMsgs2.sources.Count -eq 4)) "count=$($ownerMsgs2.sources.Count)"
        }
    }
}

# ---- 5. Report + cleanup ----------------------------------------------------
Write-Host ""
Write-Host ("{0,-40} {1}" -f "TEST", "RESULT") -ForegroundColor Cyan
Write-Host ("-" * 55)
$failed = 0
foreach ($t in $results) {
    $mark = "PASS"; $color = "Green"
    if (-not $t.Pass) { $mark = "FAIL"; $color = "Red"; $failed++ }
    Write-Host ("{0,-40} {1}" -f $t.Name, $mark) -ForegroundColor $color
    if (-not $t.Pass -and $t.Detail) { Write-Host ("  -> {0}" -f $t.Detail) -ForegroundColor Red }
}
Write-Host ""
if ($failed -eq 0) { Write-Host "PASSED: $($results.Count)/$($results.Count)" -ForegroundColor Green }
else { Write-Host "FAILED: $failed of $($results.Count) tests" -ForegroundColor Red }

Remove-Item $Cookie -Force -ErrorAction SilentlyContinue
Remove-Item $Tmp -Recurse -Force -ErrorAction SilentlyContinue
if ($startedServer) {
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    Remove-Item $BinDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "Stopped backend started by this script." -ForegroundColor Yellow
}
exit $(if ($failed -eq 0) { 0 } else { 1 })