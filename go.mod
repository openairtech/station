module github.com/openairtech/station

replace github.com/openairtech/api v0.0.0 => ../openair-api

require (
	github.com/NotifAi/serial v0.2.7
	github.com/d2r2/go-bsbmp v0.0.0-20190515110334-3b4b3aea8375
	github.com/d2r2/go-i2c v0.0.0-20181113114621-14f8dd4e89ce
	github.com/d2r2/go-logger v0.0.0-20210606094344-60e9d1233e22
	github.com/grandcat/zeroconf v1.0.0
	github.com/openairtech/api v0.0.0
	github.com/openairtech/sds011 v0.0.0-20191029135153-f4ccb629bd55
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.6.1
)

require (
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.0.0-20211118161319-6a13c67c3ce4 // indirect
	golang.org/x/sys v0.0.0-20211117180635-dee7805ff2e1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

go 1.20

replace github.com/NotifAi/serial => github.com/notifai/serial v0.2.7
