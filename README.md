# PSONI: Pub-Sub Overlay Net Interface

## Protocol Messages

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

### NATS Adapter
### Poldercast WebRTC
### Local Network Multicast

## Clients

### Groupcast