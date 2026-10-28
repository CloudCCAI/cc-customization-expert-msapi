$ErrorActionPreference = 'Stop'

if ($env:OS -ne 'Windows_NT') {
    throw 'This installer supports Windows; on macOS use tools/install.sh.'
}
$architecture = $env:PROCESSOR_ARCHITECTURE
if ($env:PROCESSOR_ARCHITEW6432) { $architecture = $env:PROCESSOR_ARCHITEW6432 }
# Read native machine metadata as well: emulated shells can report their guest architecture.
$nativeArchitecture = Get-ItemPropertyValue -LiteralPath 'Registry::HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Session Manager\Environment' -Name PROCESSOR_ARCHITECTURE -ErrorAction SilentlyContinue
if ($nativeArchitecture) { $architecture = $nativeArchitecture }
switch (([string]$architecture).ToUpperInvariant()) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { throw "Unsupported Windows architecture: $architecture" }
}
$binary = Join-Path $PSScriptRoot "bin-windows-$arch\cloudcc.exe"
if (!(Test-Path -LiteralPath $binary -PathType Leaf)) {
    throw "Bundled binary missing: $binary"
}
if (!$env:LOCALAPPDATA) { throw 'LOCALAPPDATA is not defined.' }
# The executable stays in the complete skill package so bundled assets remain accessible.
# Percent expansion cannot represent a literal percent in a generated batch command reliably.
if ($binary.Contains('%') -or $binary.Contains("`r") -or $binary.Contains("`n")) {
    throw 'The skill path cannot contain percent signs or line breaks; move it and retry.'
}
& $binary --version
if ($LASTEXITCODE -ne 0) { throw 'Bundled cloudcc failed its version check.' }
$binDir = Join-Path $env:LOCALAPPDATA 'CloudCC\bin'
$destination = Join-Path $binDir 'cloudcc.cmd'
$marker = '@rem CloudCC managed global launcher'
if (Test-Path -LiteralPath $destination) {
    if (!(Test-Path -LiteralPath $destination -PathType Leaf) -or
        !(Get-Content -LiteralPath $destination -Raw).StartsWith($marker)) {
        throw "Refusing to overwrite an unrelated file: $destination"
    }
}
foreach ($conflict in @('cloudcc.exe', 'cloudcc.bat', 'cloudcc.ps1')) {
    if (Test-Path -LiteralPath (Join-Path $binDir $conflict)) {
        throw "An existing $conflict would shadow the launcher in $binDir. Move it before installing."
    }
}
$launcher = @(
    $marker,
    '@echo off',
    'setlocal EnableExtensions DisableDelayedExpansion',
    ('"' + $binary + '" %*'),
    'set "CLOUDCC_EXIT=%errorlevel%"',
    'exit /b %CLOUDCC_EXIT%'
) -join "`r`n"
New-Item -ItemType Directory -Path $binDir -Force | Out-Null
# The Go executable handles native Windows Unicode console output. The launcher
# must not mutate the shared console code page because CMD/ConPTY may redraw and
# erase the command output when the code page is restored.
[IO.File]::WriteAllText($destination, $launcher + "`r`n", (New-Object Text.UTF8Encoding($false)))
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$entries = @($userPath -split ';' | Where-Object {
    $_ -and ![string]::Equals(
        [Environment]::ExpandEnvironmentVariables($_).Trim('"').TrimEnd('\'),
        $binDir.TrimEnd('\'), [StringComparison]::OrdinalIgnoreCase)
})
$newUserPath = (@($binDir) + $entries) -join ';'
if ($newUserPath -ne $userPath) {
    [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
}
# Also make cloudcc available immediately in this PowerShell process.
$processEntries = @($env:Path -split ';' | Where-Object {
    $_ -and ![string]::Equals(
        [Environment]::ExpandEnvironmentVariables($_).Trim('"').TrimEnd('\'),
        $binDir.TrimEnd('\'), [StringComparison]::OrdinalIgnoreCase)
})
$env:Path = (@($binDir) + $processEntries) -join ';'
cloudcc --version
if ($LASTEXITCODE -ne 0) { throw 'Global cloudcc verification failed.' }
Write-Host "Installed cloudcc ($arch): $destination"
Write-Host 'User PATH configured. This PowerShell is ready; restart other terminals and applications.'
