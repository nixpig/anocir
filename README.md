[![build](https://github.com/nixpig/brownie/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/nixpig/brownie/actions/workflows/build.yml)

# 🍪 brownie

An experimental Linux container runtime; working towards OCI Runtime Spec compliance.

> [!CAUTION]
> This is an experimental project. Feel free to have a look around, but **don't use it in anything that even resembles production**.

## Motivation

I've been learning Go for a few months now, and this seems like a reasonably sized and well-suited project to spend some time getting stuck into.

This is currently a personal project, so I'm not interested in taking code contributions. However, if you have any comments/suggestions/feedback, do feel free to leave them in [issues](https://github.com/nixpig/brownie/issues).

## Installation

### Build from source

Brownie is written in Go. You'll need the Go compiler installed to build from source.

Assuming you have a Go compiler installed...

1. `git clone git@github.com:nixpig/brownie.git`
1. `cd brownie`
1. `make build`
1. `mv tmp/bin/brownie ~/.local/bin`

## License

[MIT](https://github.com/nixpig/brownie?tab=MIT-1-ov-file#readme)
