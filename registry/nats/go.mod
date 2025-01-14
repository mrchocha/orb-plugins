module github.com/go-orb/plugins/registry/nats

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230725002816-6b0ffbf94b15
	github.com/go-orb/plugins/log/text v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/registry/tests v0.0.0-20230713091520-67e7b5a34489
	github.com/nats-io/nats.go v1.28.0
	github.com/pkg/errors v0.9.1
	golang.org/x/exp v0.0.0-20230724220655-d98519c11495
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/nats-io/nats-server/v2 v2.9.19 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/log/text => ../../log/text

replace github.com/go-orb/plugins/registry/tests => ../tests

replace github.com/go-orb/go-orb => ../../../go-orb
