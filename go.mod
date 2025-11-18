module simpleWT

go 1.25

require (
	capnproto.org/go/capnp/v3 v3.1.0-alpha.2
	github.com/go-faker/faker/v4 v4.7.0
	github.com/gofrs/uuid/v5 v5.4.0
	github.com/quic-go/quic-go v0.56.0
	github.com/quic-go/webtransport-go v0.9.0
)

require (
	github.com/colega/zeropool v0.0.0-20230505084239-6fb4a4f75381 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)

//replace capnproto.org/go/capnp/v3 => ../../github/go-capnp

replace capnproto.org/go/capnp/v3 => github.com/TheGreatSage/go-capnp/v3 v3.1.2-sage.2
