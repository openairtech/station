module github.com/openairtech/station-esp

replace github.com/openairtech/api v0.0.0 => ../openair-api

require (
	github.com/cenkalti/backoff v2.1.1+incompatible // indirect
	github.com/grandcat/zeroconf v0.0.0-20190424104450-85eadb44205c
	github.com/miekg/dns v1.1.8 // indirect
	github.com/openairtech/api v0.0.0
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.4.1
	github.com/stretchr/testify v1.2.2
	golang.org/x/net v0.0.0-20190424112056-4829fb13d2c6 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
)
