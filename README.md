# Game WebTransport Testing
A simple WebTransport setup for testing. 
This project started as simple WebTransport server to test how it behaves behind a reverse proxy.
It has gotten extended to include a basic scope to learn how everything fits together.

## WARNING: Unfinished
The scope of this project is set but the implementation hasn't been finished.
There is still bugs and the web frontend hasn't been finished. 
Most of the server is in place, the clients haven't been finished.

The plans are to finish this after a small break.

## Scope
The server should handle a few things:
- Broadcast when another client joins.
- Broadcast client chats.
- Allow clients to move on a 2D grid.
- Broadcast clients position when they move.
- When a client joins send other players current positions.
- Request an amount of 'garbage' from the client every few seconds
- Simple heartbeat / ping packet request and response.
The client:
- Respond to heartbeats
- On 'garbage' request send valid 'garbage'.
- Optionally allow for moving and chat.

## Out Of Scope
This is not a full project. 
- Authorization/Authentication are way out of scope for this. This is for WebTransport tests only.
- Databases for a similar reason. There is no persistence so no need for a DB.

You can extend a lot of features that are part of the scope to be more robust but for this project it is considered out of scope.
This wasn't an exercise in minimalism or is anything perfect. The goal was just to get it to work. 
I was learning to set most of this up for the first time so some of it is probably bad. 

## Running
To start a server either run `go run main.go` or `cmd/server.go`.

A go client is at `cmd/client.go` it can run multiple clients by passing a flag. `go run client.go -c=100`.

The `cmd/server.go` can also accept a client flag `go run server.go -c=100` to start the server then add clients.

### Web Frontend Notes
It's not finished, ran out of my original time around here.
Using Svelte and pnpm for frontend just because I've liked it on other projects.
It has some conflicts with capnp though.

### Known Issues
 - Clients crash at `-c=20` or so. 
 - Heartbeat handling on client disconnect not the best.
 - Probably others.

### Credits
Me playing with WebTransport wouldn't have started without finding: https://github.com/knervous/eqrequiem.

Some of the code either comes from there or originated there.
The lack of documentation / unclear docs on several things would have stopped me in my tracks without seeing a working example.
I try not to copy code without understanding what it does and can't think of a different or better way.

Anyway big thanks to Knervous for a lot of parts of this.