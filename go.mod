module github.com/jursonmo/practise

go 1.15

require (
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5
	github.com/asavie/xdp v0.3.3
	github.com/cilium/ebpf v0.4.0
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-mods/zerolog-rotate v1.0.2
	github.com/google/gopacket v1.1.19
	github.com/google/gops v0.3.22
	github.com/google/uuid v1.3.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/memberlist v0.2.0
	github.com/iancoleman/strcase v0.2.0
	github.com/influxdata/influxdb v1.9.5 // indirect
	github.com/influxdata/influxdb-client-go/v2 v2.5.1
	github.com/influxdata/influxdb1-client v0.0.0-20191209144304-8bf82d3c094d
	github.com/networkop/xdp-xconnect v0.0.0-20210308194118-1e1a8482c3bc
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/rs/zerolog v1.26.1
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/tevjef/go-runtime-metrics v0.0.0-20170326170900-527a54029307
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f // indirect
	github.com/vrischmann/go-metrics-influxdb v0.1.1
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.27.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/exporters/jaeger v1.3.0
	go.opentelemetry.io/otel/sdk v1.3.0
	go.opentelemetry.io/otel/trace v1.3.0
	golang.org/x/net v0.0.0-20220114011407-0dd24b26b47d // indirect
	golang.org/x/sys v0.0.0-20220111092808-5a964db01320
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/brucespang/go-tcpinfo v0.2.0 => github.com/jursonmo/go-tcpinfo v0.2.1-0.20211130062728-5c8ac4f72951
