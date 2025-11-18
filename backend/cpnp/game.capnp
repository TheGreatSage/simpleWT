using Go = import "go.capnp";
@0xb544471b4cf6fddd;
$Go.package("cpnp");
$Go.import("simpleWT/cpnp");

struct Player {
    id @0 :Text;
    # User ID as UTF8 string
    name @1 :Text;
    x @2 :Int32;
    y @3 :Int32;
}

struct GameBroadcastConnect {
    player @0 :Player;
    connected @1 :Bool;
    # Connect or disconnect
}

struct GameBroadcastChat {
    # Someone sent a chat message.
    name @0 :Text;
    # Who, maybe also send as whole player instead?
    text @1 :Text;
    # What they sent.
}

struct GameBroadcastPlayerMove {
    who @0 :Player;
}

struct GameServerGarbage {
    amount @0 :UInt32;
    # Amount of garbage to send per message
    # Probably can't actually send a whole lot here.
    per @1 :UInt8;
    # How many per second

    base @2 :Data;
    # Base of the message
    # sha1 (base + per.N)
}

struct GameServerGarbageAck {
    # Acknowledge a garbage message 
    ack @0 :UInt32;
    # Number that was just acknowledged
}

struct GameServerPlayers {
    # List of players
    players @0 :List(Player);
}

struct GameClientChat {
    # When a client wants to chat.
    text @0 :Text;
}


struct GameClientMoved {
    # When a client moves
    # -1, 0, 1
    x @0 :Int8;
    y @1 :Int8;
}

struct GarbageData {
    # :List(Data) is horrible to work with 
    data @0 :Data;
}

struct GameClientGarbage {
    # Garbage Packet
    hash @0 :List(GarbageData);
    # sha1 (base + per.N)
}

