module simpleWT

go 1.25

require (
	capnproto.org/go/capnp/v3 v3.1.0-alpha.1
	github.com/go-faker/faker/v4 v4.7.0
	github.com/gofrs/uuid/v5 v5.3.2
	github.com/quic-go/quic-go v0.55.0
	github.com/quic-go/webtransport-go v0.9.0
)

require (
	github.com/colega/zeropool v0.0.0-20230505084239-6fb4a4f75381 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
)

//replace capnproto.org/go/capnp/v3 => ../../github/go-capnp

replace capnproto.org/go/capnp/v3 => github.com/TheGreatSage/go-capnp/v3 v3.1.2-sage
