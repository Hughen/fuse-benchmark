# Go FUSE Performance

> Go version: 1.15.3 linux/amd64

## Conclusions Table

| Library                                     | Version           | block size | Performance  |
| ------------------------------------------- | ----------------- | ---------- | ------------ |
| [hanwen](https://github.com/hanwen/go-fuse) | v2.0.3            | 128        | 5324.75MiB/s |
| [jacobsa](https://github.com/jacobsa/fuse)  | latest(`36e01f1`) | 128        | 3659.92MiB/s |

[Hanwen](https://github.com/hanwen/go-fuse) can support up to 7.28 protocol, and [Jacobsa](https://github.com/jacobsa/fuse) can only support 7.12.
Starting from the Linux kernel version 2.6.35, the 7.14 protocol began to support splice. Starting from the 7.20 protocol, the `FUSE_AUTO_INVAL_DATA` function has been supported. If necessary, this feature can be disabled to let the kernel trust the FUSE data. This is very important for network file systems, it can greatly increase FUSE IOPS.

But the Hanwen design is very obscure, and there are no necessary comments in the code, which is very unfriendly to novices. In this regard, Jacobsa has to do a lot better

### Tools

The testing tools used is read-only and direct IO, and the code is located in the `tools/bench-read.go`
