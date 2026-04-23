# Installing bd

`bd` ships as a single statically-linked binary. The install script is the path of least resistance; building from source is the path you want if you plan to contribute.

## Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/signalnine/bd/main/scripts/install.sh | bash
```

The script detects platform/arch, downloads the matching archive from GitHub releases, verifies checksums, and installs to `~/.local/bin/bd` (or `/usr/local/bin/bd` if writable). Supported targets: Linux (amd64, arm64), macOS (Intel, Apple Silicon), Windows (amd64, arm64), Android/Termux (arm64), FreeBSD (amd64).

Make sure `~/.local/bin` is on your `PATH` if you went the user-local route.

## Build from source

Requires Go 1.25+ and CGO (embedded Dolt needs it).

```bash
git clone https://github.com/signalnine/bd
cd bd
make install   # builds and installs to ~/.local/bin/bd
```

On macOS you'll also need ICU4C from Homebrew (`brew install icu4c`); the Makefile wires up the include/lib paths for you.

## Mise

```bash
mise install github:signalnine/bd@latest
mise use -g github:signalnine/bd
```

Pulls the latest GitHub release, no Go toolchain needed.

## Verify

```bash
bd version
```

## Upgrading

Re-run whatever install method you originally used.

- Install script: re-run the `curl | bash` one-liner.
- From source: `git pull && make install`.
- Mise: `mise up github:signalnine/bd`.

## Uninstalling

```bash
rm "$(command -v bd)"
```

Optionally remove `.bd/` directories from your projects to delete issue databases.
