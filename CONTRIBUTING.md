# Contributing to Coddiwomple

## Licence

Coddiwomple is licensed under the [Apache 2.0 licence](LICENCE).

## Building

To build:
```bash
dep ensure
make
```

to re-build easily
```bash
make clean && make
```

Of course, `go build github.com/tetratelabs/mcc/cmd/cw` also works (assuming `dep ensure` has been run)
