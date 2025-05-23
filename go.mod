module github.com/elastic/elastic-agent-system-metrics

go 1.17

require (
	github.com/elastic/elastic-agent-libs v0.2.11
	github.com/elastic/go-licenser v0.4.0
	github.com/elastic/go-structform v0.0.9
	github.com/elastic/go-sysinfo v1.7.1
	github.com/elastic/go-windows v1.0.1
	github.com/elastic/gosigar v0.14.2
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901
	github.com/magefile/mage v1.13.0
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/shirou/gopsutil/v3 v3.21.12
	github.com/stretchr/testify v1.8.2
	go.elastic.co/go-licence-detector v0.5.0
	golang.org/x/sys v0.6.0
)

require (
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/elastic/go-ucfg v0.8.5 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/gobuffalo/here v0.6.0 // indirect
	github.com/google/licenseclassifier v0.0.0-20200402202327-879cb1424de0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/karrick/godirwalk v1.15.8 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/markbates/pkger v0.17.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.elastic.co/ecszap v1.0.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	howett.net/plist v1.0.0 // indirect
)

replace github.com/shirou/gopsutil/v3 => github.com/easyops-cn/gopsutil/v3 v3.24.9
