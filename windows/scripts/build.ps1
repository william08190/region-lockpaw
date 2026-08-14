param(
    [ValidatePattern('^\d+\.\d+\.\d+$')]
    [string]$Version = '1.0.0'
)

$ErrorActionPreference = 'Stop'
$RootDirectory = Split-Path -Parent $PSScriptRoot
$FileVersion = "$Version.0"
$BuildDirectory = Join-Path $RootDirectory 'build\windows-amd64'
$DistDirectory = Join-Path $RootDirectory 'dist'
$AppExecutable = Join-Path $BuildDirectory 'RegionLockpaw.exe'
$SetupExecutable = Join-Path $DistDirectory "RegionLockpawSetup-$Version-x64.exe"
$ChecksumFile = Join-Path $DistDirectory 'SHA256SUMS.txt'

New-Item -ItemType Directory -Force -Path $BuildDirectory, $DistDirectory | Out-Null
Remove-Item -Force -ErrorAction SilentlyContinue $AppExecutable, $SetupExecutable, $ChecksumFile

Push-Location $RootDirectory
try {
    go test ./internal/...
    if ($LASTEXITCODE -ne 0) { throw 'Internal tests failed.' }

    go vet ./internal/geometry
    if ($LASTEXITCODE -ne 0) { throw 'Geometry vet failed.' }

    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go vet ./internal/platform ./cmd/region-lockpaw
    if ($LASTEXITCODE -ne 0) { throw 'Windows vet failed.' }

    Push-Location (Join-Path $RootDirectory 'cmd\region-lockpaw')
    try {
        go run github.com/tc-hib/go-winres@v0.3.3 simply `
            --arch amd64 `
            --out rsrc `
            --manifest gui `
            --product-version $FileVersion `
            --file-version $FileVersion `
            --file-description 'Region Lockpaw' `
            --product-name 'Region Lockpaw' `
            --copyright 'Copyright (c) 2026 Region Lockpaw contributors' `
            --original-filename 'RegionLockpaw.exe' `
            --icon (Join-Path $RootDirectory 'assets\RegionLockpaw.png')
        if ($LASTEXITCODE -ne 0) { throw 'Windows resource generation failed.' }
    }
    finally {
        Pop-Location
    }

    go build `
        -trimpath `
        -ldflags "-s -w -H=windowsgui -X main.version=$Version" `
        -o $AppExecutable `
        ./cmd/region-lockpaw
    if ($LASTEXITCODE -ne 0) { throw 'Windows application build failed.' }

    $MakeNsisCommand = Get-Command makensis.exe -ErrorAction SilentlyContinue
    if ($MakeNsisCommand) {
        $MakeNsisPath = $MakeNsisCommand.Source
    }
    else {
        $Fallback = Join-Path ${env:ProgramFiles(x86)} 'NSIS\makensis.exe'
        if (Test-Path $Fallback) {
            $MakeNsisPath = $Fallback
        }
        else {
            throw 'makensis.exe was not found. Install NSIS 3.x first.'
        }
    }

    & $MakeNsisPath `
        "/DVERSION=$Version" `
        "/DFILE_VERSION=$FileVersion" `
        "/DAPP_EXE=$AppExecutable" `
        "/DOUTPUT_EXE=$SetupExecutable" `
        (Join-Path $RootDirectory 'installer\RegionLockpaw.nsi')
    if ($LASTEXITCODE -ne 0) { throw 'NSIS installer build failed.' }

    $Hashes = Get-FileHash -Algorithm SHA256 $AppExecutable, $SetupExecutable
    $ChecksumLines = $Hashes | ForEach-Object {
        "$($_.Hash.ToLowerInvariant())  $([System.IO.Path]::GetFileName($_.Path))"
    }
    $ChecksumLines | Set-Content -Encoding ascii $ChecksumFile
    $Hashes | Format-Table -AutoSize
    Write-Host "Checksums: $ChecksumFile"
}
finally {
    Pop-Location
}
