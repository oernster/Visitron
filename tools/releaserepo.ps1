# Answers the repository this checkout's releases come from, as owner/name,
# read off its origin remote (Amendment 15). build.ps1 hands it to the
# application through -ldflags -X, so no account is ever written into the
# source and a fork checks its own releases. It answers an empty string when
# there is no git, no origin or an origin that is not on GitHub; the build then
# has no update source and the window says so.
param(
    # The remote address to read; the checkout's own origin when not given.
    [string]$Origin
)

if (-not $PSBoundParameters.ContainsKey('Origin')) {
    $root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
    try {
        $Origin = (& git -C $root remote get-url origin 2>$null | Out-String).Trim()
        if ($LASTEXITCODE -ne 0) { $Origin = '' }
    } catch {
        $Origin = ''
    }
}

# https://github.com/owner/name(.git) and git@github.com:owner/name(.git).
if ($Origin -match '^(?:https://github\.com/|git@github\.com:)([A-Za-z0-9-]+)/([A-Za-z0-9._-]+?)(?:\.git)?/?$') {
    "$($Matches[1])/$($Matches[2])"
} else {
    ''
}
