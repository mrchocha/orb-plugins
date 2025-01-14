module github.com/go-orb/plugins/registry/mdns

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230725002816-6b0ffbf94b15
	github.com/go-orb/plugins/log/text v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/registry/tests v0.0.0-20230713091520-67e7b5a34489
	github.com/google/uuid v1.3.0
	github.com/miekg/dns v1.1.55
	github.com/stretchr/testify v1.8.4
	golang.org/x/net v0.12.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/exp v0.0.0-20230724220655-d98519c11495 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/tools v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/log/text => ../../log/text

replace github.com/go-orb/plugins/registry/tests => ../tests

replace github.com/go-orb/go-orb => ../../../go-orb
