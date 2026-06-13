module github.com/jeremygprawira/herr/adapter/zap

go 1.26.1

replace github.com/jeremygprawira/herr => ../../

require (
	github.com/jeremygprawira/herr v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.28.0
)

require go.uber.org/multierr v1.10.0 // indirect
