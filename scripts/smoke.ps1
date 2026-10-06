<#
.SYNOPSIS
    PowerShell port of scripts/smoke.sh for machines without bash/jq/Mailpit
    (for example a Windows box where Docker Desktop is unavailable).

.DESCRIPTION
    Exercises the documented business flow against a running API: create a
    workspace, add catalog and stock, run an order through its lifecycle, verify
    the stock ledger, the report summary and the CSV export, save a low-stock
    policy and request a password reset. The Mailpit assertion only runs when
    Mailpit is reachable; otherwise the step is reported as skipped, so the
    script stays useful on a host without an SMTP inbox.

.EXAMPLE
    powershell -File scripts\smoke.ps1
    powershell -File scripts\smoke.ps1 -BaseUrl http://localhost:8090
#>
[CmdletBinding()]
param(
    [string]$BaseUrl = $(if ($env:BASE_URL) { $env:BASE_URL } else { 'http://localhost:8080' }),
    [string]$MailpitUrl = $(if ($env:MAILPIT_URL) { $env:MAILPIT_URL } else { 'http://localhost:8025' }),
    [int]$MailpitWaitSeconds = 15
)

$ErrorActionPreference = 'Stop'
$script:checks = 0
$script:failures = 0

function Assert([bool]$Condition, [string]$Message) {
    $script:checks++
    if ($Condition) { Write-Host "  ok   $Message" }
    else { $script:failures++; Write-Host "  FAIL $Message" }
}

function Api([string]$Method, [string]$Path, $Body, [string]$Token) {
    $headers = @{}
    if ($Token) { $headers['Authorization'] = "Bearer $Token" }
    $request = @{ Method = $Method; Uri = "$BaseUrl$Path"; Headers = $headers; TimeoutSec = 30 }
    if ($null -ne $Body) {
        $request['ContentType'] = 'application/json'
        $request['Body'] = ($Body | ConvertTo-Json -Depth 6 -Compress)
    }
    return Invoke-RestMethod @request
}

function Raw([string]$Path, [string]$Token) {
    $headers = @{}
    if ($Token) { $headers['Authorization'] = "Bearer $Token" }
    return (Invoke-WebRequest -Uri "$BaseUrl$Path" -Headers $headers -TimeoutSec 30 -UseBasicParsing).Content
}

Write-Host "smoke: $BaseUrl"
$health = Api 'GET' '/healthz' $null $null
Assert ($health.status -eq 'ok') '/healthz reports ok'
$ready = Api 'GET' '/readyz' $null $null
Assert ($ready.status -eq 'ok') '/readyz reports ok'
Assert ((Raw '/') -match 'StockPilot') 'console index is served'
Assert ((Raw '/reset-password') -match 'StockPilot') 'reset-password page is served'
Assert ((Raw '/app.js') -match 'refreshToken') 'app.js is served'
$favicon = Invoke-WebRequest -Uri "$BaseUrl/favicon.svg" -TimeoutSec 20 -UseBasicParsing
Assert ($favicon.StatusCode -eq 200 -and $favicon.Headers['Content-Type'] -like 'image/svg*') 'favicon is served as svg'

$email = "smoke-$(Get-Date -Format yyyyMMddHHmmss)-$PID@example.com"
$org = Api 'POST' '/organizations' @{ name = 'Smoke Workspace'; owner_email = $email; owner_name = 'Owner'; owner_password = 'password123' } $null
$orgId = $org.id
Assert ([bool]$orgId) 'workspace created'
$prefix = "/organizations/$orgId"
$token = (Api 'POST' "$prefix/sessions" @{ email = $email; password = 'password123' } $null).access_token
Assert ([bool]$token) 'signed in with the owner credentials'

$warehouse = (Api 'POST' "$prefix/warehouses" @{ name = 'Main warehouse' } $token).id
$product = (Api 'POST' "$prefix/products" @{ name = 'Demo widget' } $token).id
$sku = (Api 'POST' "$prefix/products/$product/skus" @{ code = 'WIDGET-01'; name = 'Widget'; price_cents = 1299 } $token).id
Assert ([bool]$warehouse -and [bool]$sku) 'catalog created'

$stockPath = "$prefix/warehouses/$warehouse/skus/$sku/stock"
$received = Api 'POST' "$stockPath/receive" @{ quantity = 10; idempotency_key = 'smoke-receive-1' } $token
Assert ($received.on_hand -eq 10 -and $received.available -eq 10) 'received 10 units'

$order = Api 'POST' "$prefix/orders" @{ idempotency_key = 'smoke-order-1'; lines = @(@{ warehouse_id = $warehouse; sku_id = $sku; quantity = 2 }) } $token
Assert ($order.status -eq 'pending' -and $order.total_cents -eq 2598) 'order reserved at the catalog price'
$reserved = Api 'GET' $stockPath $null $token
Assert ($reserved.on_hand -eq 10 -and $reserved.reserved -eq 2 -and $reserved.available -eq 8) 'stock shows the reservation'

$state = $order
foreach ($step in @('confirm', 'pay', 'ship', 'complete')) {
    $state = Api 'POST' "$prefix/orders/$($order.id)/$step" @{} $token
}
Assert ($state.status -eq 'completed') 'order walked pending -> completed'
$completed = Api 'GET' $stockPath $null $token
Assert ($completed.on_hand -eq 8 -and $completed.reserved -eq 0 -and $completed.available -eq 8) 'confirmation deducted the reserved stock'

$summary = Api 'GET' "$prefix/reports/summary?low_stock_threshold=8" $null $token
Assert ($summary.orders.completed -eq 1 -and $summary.orders.confirmed_cents -eq 2598 -and $summary.inventory.low_stock -eq 1) 'report summary matches the flow'
Assert ((Raw "$prefix/exports/stocks/csv" $token) -match $sku) 'stock CSV export contains the SKU'

$alert = Api 'PUT' "$prefix/alerts/stock" @{ threshold = 8; enabled = $true } $token
Assert ($alert.threshold -eq 8 -and $alert.enabled) 'low-stock policy saved'
$reset = Api 'POST' "$prefix/sessions/password-reset" @{ email = $email } $null
Assert ([bool]$reset.message) 'password reset accepted for an existing account'

$mailpitUp = $false
try { $null = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/messages" -TimeoutSec 5; $mailpitUp = $true } catch { }
if ($mailpitUp) {
    $delivered = $false
    for ($i = 0; $i -lt $MailpitWaitSeconds; $i++) {
        try {
            $messages = Invoke-RestMethod -Uri "$MailpitUrl/api/v1/messages" -TimeoutSec 5
            $match = $messages.messages | Where-Object {
                $_.Subject -eq 'Reset your workspace password' -and ($_.To | Where-Object { $_.Address -eq $email })
            }
            if ($match) { $delivered = $true; break }
        }
        catch { }
        Start-Sleep -Seconds 1
    }
    Assert $delivered 'reset email delivered to Mailpit'
}
else {
    Write-Host "  skip Mailpit not reachable at $MailpitUrl - email delivery not asserted"
}

$passed = $script:checks - $script:failures
Write-Host "smoke: $passed/$($script:checks) checks passed"
if ($script:failures -gt 0) { exit 1 }
