# Psycho: P2P-enabling PubSub protocol and libraries #

Psycho lets your app do pubsub, but in a weird way.

- [NATS](https://docs.nats.io/nats-protocol/nats-protocol) and [Redis](https://redis.io/topics/protocol)-inspired: text-based interface protocol.
- [Go proverb](https://go-proverbs.github.io/)-inspired: tiny interface size.
- IPFS's [libp2p](https://libp2p.io/)-inflicted: "hopefully not reinventing that" feverish mantra.
- Swappable overlay network implementations.
- P2P as in People-to-People.

## Protocol ##

| OP Name | Sent By | Description|Syntax|
|---------|---------|------------|------|
|INFO|Server|First message sent to the client|`INFO {["<name>":<value>],...}`|
|SUB|Client|Subscribe to a subject|`SUB <subject>\n`|
|UNSUB|Client|Unsubscribe from a subject|`UNSUB <subject>\n`|
|PUB|Client|Publish a message to a subject|`PUB <subject> <#bytes>\n<payload>\n`|
|MSG|Server|Delivers a message payload to a subscriber|`MSG <subject> <#bytes>\n<payload>\n`|
|+OK|Server|Acknowledges well-formed protocol message|`+OK`|
|-ERR|Server|Indicates a protocol error|`-ERR <error message>`|

## Servers

### Poldercast (Global, WebRTC)

### Multicast (Local Area Network)

### NATS (Adapter)

## Apps

### Psycho Store

### Music Room

## Why? ##

Imagine, as a developer, if you had to do the thing that you normally do, but without using domain names or IP addresses, TCP or any point-to-point connections, and all you could use was a subject-based pubsub system. I bet you could still do that thing, but some things would be easier and others would be harder. I want to find out what those are.

My current belief is that P2P would be easier. Peers could discover each other by merely subscribing to a known topic. Multicasting would be easier, because the topology would be abstracted away. Being able to select the underlying pubsub implementation would probably be useful.

Congestion, flow control and backpressure would be different, interesting and harder at first.

It's an adventure!.. Or a way to go through what devs using NATS already have.

## Security ##

Hahahahahahahahahahahahahaha....

## License ##

... I'm not sure ...