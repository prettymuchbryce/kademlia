http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html#FIND_VALUE

https://en.wikipedia.org/wiki/Kademlia

https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf

Node lookup:

alpha = 3
k = 20

1. Select alpha contacts from the non-empty k-bucket closest to the search node.
   If there are fewer than alpha contacts, check adjacent buckets.
   The closestNode is noted and stored

   This list is called the short list

2. Next you send parallel, asynchronous FIND_* messages to the contacts in the shortlist.
   Each node should return up to k nodes. If the contact fails to reply it is
   removed from the shortlist

3. Next you fill the shortlist with nodes from the replies received. These should be
   sorted by being closest to the target. Next you repeat #2. Each search updates the closestNode.
   You also ignore nodes you've already contacted. This continues until either
   no node in the nodes returned is closer than closestNode OR we have accumulated
   k probed and known to be active contacts. (In the case of FIND_NODE)

   If a cycle doesn't find a closer node, then we send a FIND_* message to each
   of the k closest nodes that we have not already queried.

   At the end of this process we will have EITHER accumulated a set of k active
   contacts OR (if the message was FIND_VALUE) we may have found a data value.

   Either a set of nodes, or the value is returned.

   The shortlist will contain all nodes sorted by distance (no limit)

   A "cycle" is considered contacted alpha contacts and hearing back from them (or not)


Things that could happen from the network:
    STORE
    FIND_VALUE
    FIND_NODE
    PING

Things that will happen from initialization:
    iterativeFindNode

Things that could happen from the api:
    iterativeStore
