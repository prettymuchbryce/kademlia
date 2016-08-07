TODO:
    two kinds of message timeouts
        1. Node just doesn't respond at all
        2. Node accepts the message, but doesn't send a netmsg response
    test refresh
    test expiration
    test replication
    responseMap message timeouts
    test bad/malicious messages

Nice to have:
    loose parallelism for iterative lookups
    break store into two messages and transfer bulk of data over TCP

http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#FIND_VALUE

https://en.wikipedia.org/wiki/Kademlia

https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf
