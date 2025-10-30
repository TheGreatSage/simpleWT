export enum OpCodes {
    Unused = 0,
    
    // Utility start
    Utility,
    Heartbeat,

    //Broadcast
    Broadcasts,
    BConnect,
    BPlayerMoved,
    BChat,

    //Server
    Server,
    SGarbage,
    SPlayers,

    // Client
    Client,
    CChat,
    CMoved,
    CGarbage

}