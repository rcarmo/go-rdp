This project is a Go-based HTML5 RDP client that also uses TinyGo to compile WebAssembly modules for bitmap decompression.

## Build rules

- **Always use `make`** to run linting, building, testing, etc. If there is anything missing from the `Makefile` that you need, please add it there instead of running commands manually.
- **TinyGo is mandatory** for building distribution artifacts. The WASM module must be compiled with TinyGo (`tinygo build -target wasm -opt=z`) to keep the output under 500 KB. Standard Go WASM builds produce 2 MB+ output and must not be used for dist.
  - Install with: `brew install tinygo-org/tools/tinygo`
  - The `make build-wasm` target will error if TinyGo is not installed.