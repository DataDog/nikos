# Nikos

Nikos is a simple tool for downloading Kernel headers for multiple Linux distributions. It can be used both as a CLI application and a library.

## Support

 * Debian
 * Ubuntu
 * RHEL 
 * CentOS
 * Fedora
 * OpenSUSE Leap
 * SLES
 * Google Container Optimized OS
 * WSL2

## Usage

### On a host

`$ nikos download --output /tmp`

### Inside a container

You need to bind mount a few folders from the host:

 * Ubuntu / Debian
   `/etc/apt` (if you used a different path, you can use the `--apt-config-dir` flag)

 * RHEL / CentOS / Fedora
   - `/etc/yum.repos.d` (if you used a different path, you can use the `--yum-repos-dir` flag)
   - `/etc/pki`
   - `/etc/rhsm` (for RHEL with an active subscription)

 * OpenSUSE
   - `/etc/zypp` (if you used a different path, you can use the `--yum-repos-dir` flag)

## Building

### Requirements

Both `APT` and `Container Optimized OS` use pure Golang implementationns.

To support RPM based distributions, you need [libdnf](https://github.com/rpm-software-management/libdnf).
On Fedora, simply use `dnf install libdnf-devel`. To target machines that do not have `libdnf`, an
[omnibus](https://github.com/chef/omnibus) project is available [here](https://github.com/DataDog/omnibus-nikos).

### Compilation

`$ go build -tags dnf`

If you used the `omnibus` method described above, you should use:

`$ PKG_CONFIG_PATH=/opt/nikos/embedded/lib/pkgconfig CGO_LDFLAGS="-Wl,-rpath,/opt/nikos/embedded/lib" go build -tags dnf`

## Testing

Tests are using the [Molecule framework](https://github.com/ansible-community/molecule).

To run the tests, you can either:

 * Compile the Nikos dependencies using `omnibus` as described in [Requirements](#requirements).
 * Download a precompiled version of it [here](https://glumol.com/nikos/opt.nikos.xz). You then need to
   decompress it in your local Nikos repository folder.

To run the tests for Debian, simply run in the `tests` folder:

`$ molecule test -s debian`

You can also run the tests for `centos`, `debian`, `opensuse` and `ubuntu`.

TEST