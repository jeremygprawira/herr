module github.com/jeremygeraldprawira/herr/adapter/zap

go 1.26.1

replace github.com/jeremygeraldprawira/herr => ../../

require (
	github.com/jeremygeraldprawira/herr v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.28.0
)

require go.uber.org/multierr v1.10.0 // indirect
