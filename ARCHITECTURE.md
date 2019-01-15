# Metal Core Architecture

Metal Core is used as mediator between Metal Hammer and Metal API as well as an interface to configure the
network switch ports.

## Goals

1. Noticing new PXE servers
1. Forwarding `metal-hammer` requests from PXE server to central Metal API
1. Reporting back Metal API responses
1. Configuring network switch ports
1. Rebooting provisioned servers

## Non Goals

Persistence store.

## High level design

Each server that is going to be provisioned by `metal-hammer` needs to communicate with the central Metal API in
some way in order to fetch a proper OS image to be installed.

Since the server is started in PXE boot mode it does not, and indeed cannot, have any information about connecting to
Metal API itself. Thus a sidecar called `metal-core` is installed *next to* the server on beforehand, which is already 
connected to the central Metal API instance and therefore is capable of acting as mediator.

During PXE boot `metal-core` provides its location coordinates to the server as well as the URLs of certain kernel
and initrd images to be downloaded and started. The latter one contains the `metal-hammer` binary that is configured to be
its init process.

This way the server is able to communicate with `metal-core` to eventually fetch a proper OS image to be installed.

Additionally, `metal-core` is connected to the network switch and handles port configurations to put the
provisioned servers from PXE into their destination networks, and vice versa when freed by Metal API.
It therefore also reboots them through IPMI at proper times. 

The trigger to free a certain server is initiated by a user of Metal API, which subsequently publishes it through a
proper NSQ channel. On the other end `metal-core` consumes from that channel and handles those triggers
by putting the concerned server(s) back into PXE boot mode.

## NSQ

NSQ is chosen over NATS since it is more lightweight and comes without any message broker, which fosters scalability
and HA. It also enables message replication through topics and round-robin loadbalancing through topic channels.
