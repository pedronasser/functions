# Build script for PowerShell
$ErrorActionPreference = "Stop"

$cmd = $args[0]
Write-Host "cmd: $cmd"


function quick() {
     try {
        go build
        if (-not $?) {
            Write-Host "build failed!" # WTH, error handling in powershell sucks
            exit
        }
    } catch {
        # This try/catch thing doesn't work, the above if statement does work though
        Write-Host "build failed 2!"
        exit
    }
    ./functions
}

function build () {
    docker run --rm -v ${pwd}:/go/src/github.com/pedronasser/functions -w /go/src/github.com/pedronasser/functions iron/go:dev go build -o functions-alpine
    docker build -t pedronasser/functions:latest .
}

function run () {
    docker run --rm --name functions -it -v /var/run/docker.sock:/var/run/docker.sock -e LOG_LEVEL=debug -e "DB_URL=bolt:///app/data/bolt.db" -v $PWD/data:/app/data -p 8080:8080 pedronasser/functions
}

switch ($cmd)
{
    "quick" {quick}
    "build" { build }
    "run" {run}
    default {"Invalid command: $cmd"}
}
