using Go = import "go.capnp";
@0xbc17d12a74fd5cc3;
$Go.package("cpnp");
$Go.import("simpleWT/cpnp");

struct Heartbeat {
    unix @0 :Int64;
    # Milli seconds is fine as that is the default javascript
}