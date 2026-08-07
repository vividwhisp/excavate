# Test the Excavate auth API end to end.
# Usage:  powershell -ExecutionPolicy Bypass -File scripts/test-auth.ps1
# Auto-starts Postgres/Redis (docker) and the backend if they are not running.
# Exit code 0 = all tests passed, 1 = one or more failed.

# PowerShell 5.1 treats native stderr output as errors under "Stop", which would
# abort the script on noisy-but-harmless output (e.g. docker's status lines).
# We use "Continue" and check exit codes explicitly instead.
$ErrorActionPreference = "Continue"

function Fail($msg) {
    Write-Host "ERROR: $msg" -ForegroundColor Red
    exit 1
}

# ---- 0. Prereqs -------------------------------------------------------------
if (-not (Get-Command curl.exe -ErrorAction SilentlyContinue)) {
    Fail "curl.exe not found (Windows ships it at C:\Windows\System32\curl.exe)"
}

$Root = Split-Path -Parent $PSScriptRoot
$Backend = Join-Path $Root "backend"
$BinDir = Join-Path $Backend "bin"
$Base = "http://localhost:8080"
$Cookie = Join-Path $env:TEMP "excavate_cookies_$PID.txt"

# ---- 1. Databases -----------------------------------------------------------
Write-Host "== Starting Postgres + Redis ==" -ForegroundColor Cyan
docker compose up -d postgres redis 2>&1 | Out-Null
$ready = $false
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep 2
    $ps = docker compose ps --format "{{.Name}} {{.Status}}"
    if (($ps | Select-String "healthy").Count -ge 2) { $ready = $true; break }
}
if (-not $ready) {
    Fail "Postgres/Redis did not become healthy in 60s. Run 'docker compose ps' to diagnose."
}
Write-Host "Databases healthy." -ForegroundColor Green

# ---- 2. Backend (start only if nothing is on :8080) -------------------------
$listener = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue
$startedServer = $false
if ($listener) {
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

# ---- 3. Test harness --------------------------------------------------------
$results = @()
function Add-Result([string]$name, [bool]$pass, [string]$detail) {
    $script:results += [pscustomobject]@{ Name = $name; Pass = $pass; Detail = $detail }
}

function Test-Case([string]$name, [scriptblock]$body) {
    try {
        & $body
    } catch {
        $msg = $_.Exception.Message
        Add-Result $name $false "exception: $msg"
    }
}

function Http([string]$method, [string]$path, [string]$body = "", [switch]$UseCookie) {
    $outFile = "$env:TEMP\excavate_resp.txt"
    $bodyFile = "$env:TEMP\excavate_body.txt"
    $curlArgs = @("-s", "-o", $outFile, "-w", "%{http_code}", "-X", $method, "$Base$path")
    if ($UseCookie) { $curlArgs += @("-b", $Cookie) }
    if ($body) {
        # PowerShell 5.1 mangles embedded quotes passed to native exes, so we
        # write the JSON to a temp file and use curl's @file syntax instead.
        Set-Content -Path $bodyFile -Value $body -NoNewline -Encoding ascii
        $curlArgs += @("-H", "Content-Type: application/json", "--data-binary", "@$bodyFile")
    }
    $code = & curl.exe @curlArgs
    $resp = ""
    if (Test-Path $outFile) { $resp = Get-Content $outFile -Raw }
    return @{ code = [string]$code; body = $resp }
}

# ---- 4. Run the 8 assertions ------------------------------------------------
$email = "script-$PID@test.com"
$password = "password123"
$loginBodyFile = "$env:TEMP\excavate_login.txt"
Set-Content -Path $loginBodyFile -Value "{`"email`":`"$email`",`"password`":`"$password`"}" -NoNewline -Encoding ascii
$result = $null

Test-Case "register returns 201" {
    $r = Http "POST" "/api/auth/register" "{`"email`":`"$email`",`"password`":`"$password`"}"
    if ($r.code -ne "201") { throw "expected 201, got $($r.code)" }
    Add-Result "register returns 201" $true $r.code
}

Test-Case "cookie session works (/api/me)" {
    & curl.exe -s -c $Cookie -o NUL -X POST -H "Content-Type: application/json" --data-binary "@$loginBodyFile" "$Base/api/auth/login"
    $r = Http "GET" "/api/me" $null -UseCookie
    if ($r.code -ne "200" -or $r.body -notmatch $email) { throw "expected 200 + email, got $($r.code): $($r.body)" }
    Add-Result "cookie session works (/api/me)" $true $r.code
}

Test-Case "duplicate email -> 422" {
    $r = Http "POST" "/api/auth/register" "{`"email`":`"$email`",`"password`":`"$password`"}"
    if ($r.code -ne "422" -or $r.body -notmatch "email already registered") { throw "got $($r.code): $($r.body)" }
    Add-Result "duplicate email -> 422" $true $r.code
}

Test-Case "short password -> 422" {
    $r = Http "POST" "/api/auth/register" "{`"email`":`"x-$PID@test.com`",`"password`":`"short`"}"
    if ($r.code -ne "422" -or $r.body -notmatch "password must be at least 8 characters") { throw "got $($r.code): $($r.body)" }
    Add-Result "short password -> 422" $true $r.code
}

Test-Case "wrong password login -> 401" {
    $r = Http "POST" "/api/auth/login" "{`"email`":`"$email`",`"password`":`"wrongpass1`"}"
    if ($r.code -ne "401" -or $r.body -notmatch "invalid email or password") { throw "got $($r.code): $($r.body)" }
    Add-Result "wrong password login -> 401" $true $r.code
}

Test-Case "no cookie /api/me -> 401" {
    $r = Http "GET" "/api/me"
    if ($r.code -ne "401" -or $r.body -notmatch "unauthorized") { throw "got $($r.code): $($r.body)" }
    Add-Result "no cookie /api/me -> 401" $true $r.code
}

Test-Case "login returns 200" {
    $r = Http "POST" "/api/auth/login" "{`"email`":`"$email`",`"password`":`"$password`"}" -UseCookie
    if ($r.code -ne "200" -or $r.body -notmatch $email) { throw "got $($r.code): $($r.body)" }
    Add-Result "login returns 200" $true $r.code
}

Test-Case "logout destroys session (/api/me -> 401)" {
    $r = Http "POST" "/api/auth/logout" $null -UseCookie
    if ($r.code -ne "200") { throw "logout got $($r.code)" }
    $r2 = Http "GET" "/api/me" $null -UseCookie
    if ($r2.code -ne "401") { throw "expected 401 after logout, got $($r2.code)" }
    Add-Result "logout destroys session (/api/me -> 401)" $true "$($r.code), then $($r2.code)"
}

# ---- 5. Report + exit -------------------------------------------------------
Write-Host ""
Write-Host ("{0,-45} {1}" -f "TEST", "RESULT") -ForegroundColor Cyan
Write-Host ("-" * 60)
$failed = 0
foreach ($t in $results) {
    $mark = "PASS"
    $color = "Green"
    if (-not $t.Pass) { $mark = "FAIL"; $color = "Red"; $failed++ }
    Write-Host ("{0,-45} {1}" -f $t.Name, $mark) -ForegroundColor $color
    if (-not $t.Pass -and $t.Detail) { Write-Host ("  -> {0}" -f $t.Detail) -ForegroundColor Red }
}
Write-Host ""
if ($failed -eq 0) {
    Write-Host "PASSED: $($results.Count)/$($results.Count)" -ForegroundColor Green
} else {
    Write-Host "FAILED: $failed of $($results.Count) tests" -ForegroundColor Red
}

# ---- 6. Cleanup -------------------------------------------------------------
Remove-Item $Cookie, "$env:TEMP\excavate_resp.txt", "$env:TEMP\excavate_body.txt", "$env:TEMP\excavate_login.txt" -Force -ErrorAction SilentlyContinue
if ($startedServer) {
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    Remove-Item $BinDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "Stopped backend started by this script." -ForegroundColor Yellow
}

exit $(if ($failed -eq 0) { 0 } else { 1 })
