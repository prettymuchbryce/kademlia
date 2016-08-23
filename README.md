# kademlia
[![GoDoc](https://godoc.org/github.com/prettymuchbryce/kademlia?status.svg)](https://godoc.org/github.com/prettymuchbryce/kademlia)
[![Go Report Card](https://goreportcard.com/badge/github.com/prettymuchbryce/kademlia)](https://goreportcard.com/report/github.com/prettymuchbryce/kademlia)
[![Build Status](https://travis-ci.org/prettymuchbryce/kademlia.svg?branch=master)](https://travis-ci.org/prettymuchbryce/kademlia)

This is a Go implementation of a vanilla [Kademlia](https://en.wikipedia.org/wiki/Kademlia) DHT. The implementation is based off of a combination of the original [Kademlia whitepaper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf) and the [xlattice design specification](http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html). It does not attempt to conform to BEP-5, or any other bitorrent-specific design.

_This project has not been heavily battle-tested, and I would not recommend using it in any production environment at this time._

## Implementation characteristics
-  uses uTP for all network communication
-  supports IPv4/IPv6
-  uses a well-defined Store interface for extensibility

## TODO
- [ ] Load testing / Benchmarks
- [ ] More testing around message validation
- [ ] More testing of bad/malicious message handling
- [ ] Banning/throttling of malicious messages/nodes
- [ ] Implement UDP hole punching via NAT traversal
- [ ] Use loose parallelism for iterative lookups
- [ ] Consider breaking store into two messages and transfer bulk of data over TCP
- [ ] Implement republishing according to the xlattice design document
- [ ] Better cleanup of unanswered expected messages
