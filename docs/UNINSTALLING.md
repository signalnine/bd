# Uninstalling bd

## Remove the binary

Whichever install method you used:

- Install script / `make install`: `rm "$(command -v bd)"`
- Mise: `mise uninstall github:signalnine/bd`

## Remove issue data

Per project:

```bash
rm -rf .bd
```

This deletes the embedded Dolt database and all issue history. Back up first if you care:

```bash
bd backup init /some/path
bd backup sync
```

Or keep the raw Dolt database elsewhere:

```bash
cp -r .bd/embeddeddolt /some/safe/place
```

## Remove git hook residue (legacy installs only)

bd <= v1.0.2 installed git hook shims that break commits after the binary is gone. If you were on that version, check:

```bash
git config --get core.hooksPath   # should be unset, or pointing at .git/hooks
```

If it's pointing at `.bd/hooks` or `.bd-hooks`, unset it:

```bash
git config --unset core.hooksPath
rm -rf .bd/hooks .bd-hooks
```

(Running `bd init` once on v1.0.3+ auto-cleans this.)

## Remove global config

bd does not write outside `.bd/` in your project. No global uninstall needed beyond the binary.
