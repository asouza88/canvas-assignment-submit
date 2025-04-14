param (
    [Parameter(Mandatory=$true)]
    [ValidateSet("windows", "linux", "mac-amd64", "mac-arm64")]
    [string]$target
)

$AppName = "cal"  # Change this to your binary name
$OutputDir = "build"

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

switch ($target) {
    "windows" {
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        go build -o "$OutputDir\$AppName.exe"
    }
    "linux" {
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        go build -o "$OutputDir\$AppName"
    }
    "mac-amd64" {
        $env:GOOS = "darwin"
        $env:GOARCH = "amd64"
        go build -o "$OutputDir\$AppName-mac-amd64"
    }
    "mac-arm64" {
        $env:GOOS = "darwin"
        $env:GOARCH = "arm64"
        go build -o "$OutputDir\$AppName-mac-arm64"
    }
    default {
        Write-Host "Invalid target"
        exit 1
    }
}

