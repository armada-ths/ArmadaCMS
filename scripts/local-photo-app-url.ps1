# Prefer the active default-route adapter so virtual Docker adapters are excluded.
$photoNetwork = Get-NetIPConfiguration |
    Where-Object { $_.IPv4DefaultGateway -and $_.NetAdapter.Status -eq 'Up' } |
    Sort-Object { $_.NetIPv4Interface.InterfaceMetric } |
    Select-Object -First 1
$photoAddress = if ($null -ne $photoNetwork) {
    $photoNetwork.IPv4Address.IPAddress | Select-Object -First 1
}
else {
    $null
}
if ($photoAddress) {
    "http://${photoAddress}:8001"
}
else {
    Write-Warning 'No active LAN address found. Photo links will use localhost.'
    'http://localhost:8001'
}
