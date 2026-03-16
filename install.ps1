#Requires -Version 5.1
<#
.SYNOPSIS
    Install readme-merge on Windows.
.DESCRIPTION
    Downloads the latest release from GitHub and installs the binary.
    Set $env:GITHUB_TOKEN for private repo access.
    Set $env:INSTALL_DIR to override the default install location.
#>
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$Repo = 'phasecurve/readme-merge'
$Binary = 'readme-merge'

function Main {
    $arch = Get-Arch
    $version = Get-LatestVersion
    if (-not $version) {
        Write-Error 'Could not determine latest version'
        exit 1
    }

    $installDir = Get-InstallDir
    $asset = "${Binary}_$($version.TrimStart('v'))_windows_${arch}.zip"
    $url = "https://github.com/${Repo}/releases/download/${version}/${asset}"

    $tmpDir = New-TemporaryDirectory
    try {
        Write-Host "Installing ${Binary} ${version} (windows/${arch})"
        Get-Release -Url $url -Dest (Join-Path $tmpDir $asset)
        Expand-Archive -Path (Join-Path $tmpDir $asset) -DestinationPath $tmpDir -Force

        $binaryPath = Join-Path $tmpDir "${Binary}.exe"
        if (-not (Test-Path $binaryPath)) {
            Write-Error 'Binary not found in archive'
            exit 1
        }

        Install-Binary -Source $binaryPath -DestDir $installDir
        Confirm-Path -Dir $installDir

        Write-Host "Installed ${Binary} to $(Join-Path $installDir "${Binary}.exe")"
    }
    finally {
        Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
    }
}

function Get-Arch {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        default {
            Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
            exit 1
        }
    }
}

function Get-LatestVersion {
    $apiUrl = "https://api.github.com/repos/${Repo}/releases/latest"
    $headers = @{ 'Accept' = 'application/vnd.github+json' }
    if ($env:GITHUB_TOKEN) {
        $headers['Authorization'] = "token $env:GITHUB_TOKEN"
    }

    try {
        $response = Invoke-RestMethod -Uri $apiUrl -Headers $headers
        return $response.tag_name
    }
    catch {
        if (-not $env:GITHUB_TOKEN) {
            Write-Error 'Failed to fetch latest release. If this is a private repo, set $env:GITHUB_TOKEN.'
        }
        else {
            Write-Error "Failed to fetch latest release: $_"
        }
        exit 1
    }
}

function Get-InstallDir {
    if ($env:INSTALL_DIR) {
        return $env:INSTALL_DIR
    }
    return Join-Path $env:LOCALAPPDATA $Binary
}

function Get-Release {
    param(
        [string]$Url,
        [string]$Dest
    )

    $headers = @{}
    if ($env:GITHUB_TOKEN) {
        $headers['Authorization'] = "token $env:GITHUB_TOKEN"
    }

    try {
        Invoke-WebRequest -Uri $Url -OutFile $Dest -Headers $headers
    }
    catch {
        Write-Error "Download failed: $Url"
        exit 1
    }

    if (-not (Test-Path $Dest) -or (Get-Item $Dest).Length -eq 0) {
        Write-Error "Download produced empty file: $Url"
        exit 1
    }
}

function Install-Binary {
    param(
        [string]$Source,
        [string]$DestDir
    )

    if (-not (Test-Path $DestDir)) {
        New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
    }

    Copy-Item -Path $Source -Destination (Join-Path $DestDir "${Binary}.exe") -Force
}

function Confirm-Path {
    param([string]$Dir)

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -split ';' -contains $Dir) {
        return
    }

    Write-Host ''
    Write-Host "warning: ${Dir} is not in your PATH" -ForegroundColor Yellow
    $answer = Read-Host "Add it to your user PATH? (y/N)"
    if ($answer -eq 'y') {
        $newPath = "${userPath};${Dir}"
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        $env:Path = "${env:Path};${Dir}"
        Write-Host "Added ${Dir} to user PATH. Restart your terminal for it to take effect."
    }
    else {
        Write-Host "Skipped. Add it manually: `$env:Path += `";${Dir}`""
    }
}

function New-TemporaryDirectory {
    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tmp -Force | Out-Null
    return $tmp
}

Main
