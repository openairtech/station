module github.com/openairtech/station

replace github.com/openairtech/api v0.0.0 => ../openair-api

require (
	github.com/NotifAi/serial v0.2.3
	github.com/cenkalti/backoff v2.1.1+incompatible // indirect
	github.com/d2r2/go-bsbmp v0.0.0-20190515110334-3b4b3aea8375
	github.com/d2r2/go-i2c v0.0.0-20181113114621-14f8dd4e89ce
	github.com/d2r2/go-logger v0.0.0-20181221090742-9998a510495e
	github.com/grandcat/zeroconf v0.0.0-20190424104450-85eadb44205c
	github.com/karalabe/xgo v0.0.0-20190301120235-2d6d1848fb02 // indirect
	github.com/miekg/dns v1.1.8 // indirect
	github.com/openairtech/api v0.0.0
	github.com/openairtech/sds011 v0.0.0-20191029135153-f4ccb629bd55
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sirupsen/logrus v1.4.1
	github.com/stretchr/testify v1.2.2
	golang.org/x/net v0.0.0-20190424112056-4829fb13d2c6 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
)

go 1.13
