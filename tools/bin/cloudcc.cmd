@echo off
setlocal EnableExtensions DisableDelayedExpansion
set "CLOUDCC_ARCH=%PROCESSOR_ARCHITECTURE%"
if defined PROCESSOR_ARCHITEW6432 set "CLOUDCC_ARCH=%PROCESSOR_ARCHITEW6432%"
for /f "tokens=2,*" %%A in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PROCESSOR_ARCHITECTURE 2^>nul') do if /I "%%A"=="REG_SZ" set "CLOUDCC_ARCH=%%B"
if /I "%CLOUDCC_ARCH%"=="AMD64" goto amd64
if /I "%CLOUDCC_ARCH%"=="ARM64" goto arm64
echo cloudcc: unsupported architecture: %CLOUDCC_ARCH% 1>&2
exit /b 1
:amd64
set "CLOUDCC_BINARY=%~dp0..\bin-windows-amd64\cloudcc.exe"
goto run
:arm64
set "CLOUDCC_BINARY=%~dp0..\bin-windows-arm64\cloudcc.exe"
:run
if not exist "%CLOUDCC_BINARY%" (
    echo cloudcc: bundled binary missing: "%CLOUDCC_BINARY%" 1>&2
    exit /b 1
)
"%CLOUDCC_BINARY%" %*
exit /b %errorlevel%
