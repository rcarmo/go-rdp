# RDP HTML5 Client

> ⚠️ **EXPERIMENTAL SOFTWARE** ⚠️
>
> This project is a proof-of-concept and experimental implementation. It is **NOT** intended for production use. The RDP protocol implementation is incomplete, may contain bugs, and has not undergone security auditing. Use at your own risk and only in controlled development/testing environments.

A browser-based Remote Desktop Protocol (RDP) client built with Go and WebAssembly.

## Documentation

Long-form documentation lives in `docs/`:

- [Docs index](docs/index.md)
- [Architecture](docs/architecture.md)
- [Configuration](docs/configuration.md)
- [Debugging](docs/debugging.md)

## Why

I was getting _really_ tired of not having a modern RDP client that works in a browser without extraneous dependencies and overhead (yes, I am looking at you, Guacamole), so I found [this project](https://github.com/kulaginds/rdp-html5) and decided to modernize it, fix it up, and, since I wanted to have full control over all the features, dependencies and implementation details, gradually remove all the GPL-licensed code so I can re-release it under MIT.

This project aims to provide a simple, open-source RDP client that has a minimal footprint, can replace Guacamole for most basic use cases, and should run in any modern web browser without plugins.

## Features

- **TLS Support**: TLS 1.2+ encryption for transport security
- **Basic RDP Protocol**: Core RDP functionality tested primarily against XRDP on Linux
- **Web Interface**: HTML5/JavaScript client with canvas rendering
- **WebAssembly**: RLE bitmap decompression via WASM module
- **Environment Configuration**: Configuration via environment variables

## Limitations

- **NLA / `CredSSP`**: Network Level Authentication support is incomplete
- **Windows Compatibility**: Primarily tested with XRDP; Windows RDP servers may not work
- **Graphics**: Only basic bitmap updates supported; no RemoteFX or H.264
- **Clipboard/Audio/Printing**: Not implemented
- **Virtual Channels**: Partial implementation only
- **Security**: Not audited; do not use with sensitive systems

For details on architecture, configuration, and debugging, see the docs links above.
