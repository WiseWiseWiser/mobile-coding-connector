# Container Overlay Rootfs

## Background

The boot container maps a host directory as the container's root filesystem using Podman's `--rootfs` flag. The rootfs is extracted from `debian:bookworm-slim` and stored at `script/sandbox/boot/root/`.

## Problem: Permission Denied with `--rootfs` on macOS

On macOS, Podman runs containers inside a Linux VM. Host directories are shared into the VM via **virtiofs**. When using plain `--rootfs /path`, the rootfs directory is mounted directly through virtiofs.

This caused `dpkg` and `apt-get` to fail with "Permission denied" errors when installing packages (e.g. `git`) inside the container:

```
dpkg: error processing archive ... (--unpack):
 error creating directory './usr/share/doc/perl-modules-5.36': Permission denied
dpkg: error while cleaning up:
 unable to remove newly-extracted version of '...': Permission denied
```

### Root Cause

1. **Ownership mismatch**: Files extracted with `tar` on macOS are owned by the local user. Inside the container, Podman's rootless user namespace maps UID 0 (root) to this host user. However, system processes like `_apt` (used by `apt-get` for downloads) map to different UIDs and can't access directories with restrictive permissions (e.g. `drwx------`).

2. **dpkg creates files with 000 permissions**: During package installation, dpkg creates temporary `.dpkg-new` files/directories with mode `0000` as part of its secure extraction process. Through virtiofs, these permissions are applied literally on the host filesystem, making the files inaccessible to everyone — including root inside the container.

3. **virtiofs limitations**: The virtiofs filesystem layer between macOS and the Podman VM doesn't fully support all Linux filesystem semantics that dpkg relies on (atomic renames with restrictive intermediate permissions, `fchown`, etc.).

### Attempted fixes that didn't work

- `chmod -R u+w` — only gives write permission to the owner (root), but `_apt` still can't access
- `chmod -R a+rwX` — fixes existing files, but dpkg creates new 000-permission files during installation that immediately become inaccessible

## Solution: Overlay Mode (`--rootfs /path:O`)

Changed from `--rootfs /path` to `--rootfs /path:O`.

The `:O` suffix tells Podman to create an **overlay filesystem** on top of the rootfs:

- **Lower layer** (read-only): the host rootfs directory, shared via virtiofs
- **Upper layer** (read-write): a native directory inside the Podman VM

All writes inside the container go to the native upper layer, completely bypassing virtiofs for write operations. This means:

- `dpkg` can create files with any permissions (000, etc.) without issues
- `apt-get` / `_apt` can write to cache directories normally
- The host rootfs directory remains clean and unmodified
- Package installations work correctly

### Trade-offs

- The overlay upper layer lives inside the Podman VM, not on the host. Packages installed inside the container persist as long as the container exists, but are lost when the container is removed.
- The server binary is copied into the host rootfs (lower layer) before container creation, so it's always up to date.
- Volume mounts (`-v`) for `/root` and `/root/.ai-critic` still work as before, overlaying specific paths on top of the overlay.
