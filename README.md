# Uniclip - Universal Clipboard [![Release](https://img.shields.io/github/v/release/robinhickmann/uniclip)](https://github.com/robinhickmann/uniclip/releases/latest) [![License](https://img.shields.io/github/license/robinhickmann/uniclip)](https://github.com/robinhickmann/uniclip/blob/main/LICENSE)

A simple and flexible universal clipboard for all platforms.
Uniclip is licensed under the MIT license.

You can run the server in the same binary, on your pc or a dedicated server.

Some use cases include:

- Shared VM host and guest clipboard
- Shared clipboard for devices in your local network
- Shared clipboards for temporary use

## Usage

Run this to start a new clipboard:

```sh
uniclip
```

Example output:

```text
Starting a new clipboard!
Run `uniclip 192.168.86.24:51607` to join this clipboard

```

Just enter what it says (`uniclip 192.168.86.24:51607`) on your other device with Uniclip installed and hit enter. That's it! Now you can copy from one device and paste on the other.

You can even have multiple devices joined to the same clipboard (just run that same command on the new device).

```text
uniclip -h

Usage: uniclip [--secure/-s] [--debug/-d] [ <address> | --help/-h ]
Examples:
   uniclip                                   # start a new clipboard
   uniclip -p 53701                          # start a new clipboard on the provided port
   uniclip 192.168.86.24:53701               # join the clipboard at 192.168.86.24:53701
   uniclip -d                                # start a new clipboard with debug output
   uniclip -d --secure 192.168.86.24:53701   # join the clipboard with debug output and enable encryption
Running just `uniclip` will start a new clipboard.
It will also provide an address with which you can connect to the same clipboard with another device.
```

_Note: By default uniclip binds to all interfaces meaning you can use any networking setup to reach it. However, It's not recommended to open the uniclip server to the internet since there's no authentication. Use any type of vpn instead._

## Installing

### Linux

Just grab a precompiled binary from [releases](https://github.com/quackduck/uniclip/releases)

_Install script coming soon_

### Windows

Just grab a precompiled binary from [releases](https://github.com/quackduck/uniclip/releases)

_Windows installer coming soon_

### MacOS

Just grab a precompiled binary from [releases](https://github.com/quackduck/uniclip/releases)

_Might add brew in the future_

### Android

Get an executable from [releases](https://github.com/quackduck/uniclip/releases) and install to `$PREFIX/usr/bin/uniclip`

Install the Termux app and Termux:API app from the Play Store.
Then, install the Termux:API package from the command-line (in Termux) using:

```sh
pkg install termux-api
```

_Might add apk file in the future_

### iOS

I cannot afford to make an iOS app for this silly little utility. Feel free to do so yourself. If you do, please let me know!

## Uninstalling

Uninstalling Uniclip is very easy. If you used a package manager, use it's uninstall feature. If not, just delete the Uniclip binary:

On Linux or macOS, delete `/usr/local/bin/uniclip`  
On Windows, delete it from where you installed it  
On Android, delete it from `$PREFIX/usr/bin/uniclip`

## Any other business

Have a question, idea, suggestion or bug report? Head over to [Issues](https://github.com/robinhickmann/uniclip/issues) and create one. Use appropriate tags if you can.

Contributions are happily welcomed, all I ask is for you to use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/).

## Security

If you believe you've found a security vulnerability that affects uniclip.

Please do not post it publicly and instead, open the GitHub [Security](https://github.com/robinhickmann/uniclip/security) tab. Then click the "Report a vulnerability" button.
