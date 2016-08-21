go-kademlia
=======
This is a Go implementation of a vanilla Kademlia DHT. The implementation is based off a combination of the original (Kademlia whitepaper)[https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf] and the (xlattice design specification)[http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html]. It does not attempt to conform to BEP-5, or any other bitorrent-specific design.

go-kademlia uses the uTP protocol for all network communication, and supports both IPv4 and IPv6.

go-kademlia uses a well-defined Store interface for extensibility. This allows applications to decide the best strategy to structure, store, replicate, and expire their data.

This project has not been heavily battle-tested, and I would not recommend using it in any sort of production environment at this time.

Some things that could improve in this project:
- [ ] Load testing / Benchmarks
- [ ] More testing around message validation
- [ ] More testing of bad/malicious message handling
- [ ] Banning/throttling of malicious messages/nodes
- [ ] Implement UDP hole punching via NAT traversal
- [ ] Use loose parallelism for iterative lookups
- [ ] Consider breaking store into two messages and transfer bulk of data over TCP
- [ ] Implement republishing according to the xlattice design document
- [ ] Better cleanup of unanswered expected messages
