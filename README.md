# Whisper

Whisper is a simple demonstration of peer to peer applications using gRPC.

Each server is intialized with bootnodes that can be used to relay messages. Servers only relay one level deep into the network graph.

## Install

Whisper requires Go v1.10 or higher and a valid Go workspace.

To install whisper:

`go get -u github.com/pokstad/whisper/whisper`

## Usage

Start servers like so:

```
whisper -bind 127.0.0.1:3000 -alias "paco" server &
whisper -bind 127.0.0.1:3001 -alias "tobes" -bootnodes 127.0.0.1:3000 server &
whisper -bind 127.0.0.1:3002 -alias "pooks" -bootnodes 127.0.0.1:3001 server &

```

Run clients like so:

`whisper -relay 127.0.0.1:3001 -alias paco -message hi client`
