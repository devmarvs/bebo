# Testing Guide

## Golden HTTP tests
Golden files live under `testdata/` in each package.
Update golden files with:
```sh
BEBO_UPDATE_GOLDEN=1 go test ./...
```

## Fuzz tests
Run fuzz targets for router and render:
```sh
go test ./router -fuzz=FuzzRouterMatch -fuzztime=10s
go test ./render -fuzz=FuzzTemplateNameFS -fuzztime=10s
```

## Benchmarks
Use the stable harness:
```sh
go test ./bench -bench=. -benchmem
```
Package-specific benchmarks remain in their respective packages.
