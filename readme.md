# Go-Akt

[![build](https://img.shields.io/github/actions/workflow/status/Tochemey/goakt/build.yml?branch=main)](https://github.com/Tochemey/goakt/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/tochemey/goakt.svg)](https://pkg.go.dev/github.com/tochemey/goakt)
[![codecov](https://codecov.io/gh/Tochemey/goakt/branch/main/graph/badge.svg?token=J0p9MzwSRH)](https://codecov.io/gh/Tochemey/goakt)
[![Go Report Card](https://goreportcard.com/badge/github.com/tochemey/goakt)](https://goreportcard.com/report/github.com/tochemey/goakt)

Distributed [Go](https://go.dev/) actor framework to build reactive and distributed system in golang using
_**protocol buffers as actor messages**_.

GoAkt is highly scalable and available when running in cluster mode. It comes with the necessary features require to
build a distributed actor-based system without sacrificing performance and reliability. With GoAkt, you can instantly create a fast, scalable, distributed system
across a cluster of computers.

If you are not familiar with the actor model, the blog post from Brian Storti [here](https://www.brianstorti.com/the-actor-model/) is an excellent and short introduction to the actor model.
Also, check reference section at the end of the post for more material regarding actor model.

## Table of Content

- [Design Principles](#design-principles)
- [Features](#features)
    - [Actors](#actors)
    - [Passivation](#passivation)
    - [Actor System](#actor-system)
    - [Behaviors](#behaviors)
    - [Mailbox](#mailbox)
    - [Dead Letters](#dead-letters)
    - [Messaging](#messaging)
    - [Scheduler](#scheduler)
    - [Stashing](#stashing)
    - [Remoting](#remoting)
    - [Clustering](#cluster)
    - [Observability](#observability)
        -  [Logging](#logging)
        - [Tracing](#tracing)
    - [Testkit](#testkit)
- [Use Cases](#use-cases)
- [Installation](#installation)
- [Clustering](#clustering)
    - [Operation Guides](#operations-guide)
    - [Discovery Providers](#built-in-discovery-providers)
        - [Kubernetes](#kubernetes-discovery-provider-setup)
        - [mDNS](#mdns-discovery-provider-setup)
        - [NATS](#nats-discovery-provider-setup)
        - [Domain Name Discovery](#dns-provider-setup)
- [Examples](#examples)
- [Contribution](#contribution)
    - [Local Test and Linter](#test--linter)
- [Benchmark](#benchmark)

## Design Principles

This framework has been designed:

- to be very simple - it caters for the core component of an actor framework as stated by the father of the actor framework [here](https://youtu.be/7erJ1DV_Tlo).
- to be very easy to use.
- to have a clear and defined contract for messages - no need to implement/hide any sort of serialization.
- to make use existing battle-tested libraries in the go ecosystem - no need to reinvent solved problems.
- to be very fast.
- to expose interfaces for custom integrations rather than making it convoluted with unnecessary features.

## Features

### Actors

The fundamental building blocks of Go-Akt are actors.

- They are independent, isolated unit of computation with their own state.
- They can be _long-lived_ actors or be _passivated_ after some period of time that is configured during their
  creation. Use this feature with care when dealing with persistent actors (actors that require their state to be persisted).
- They are automatically thread-safe without having to use locks or any other shared-memory synchronization
  mechanisms.
- They can be stateful and stateless depending upon the system to build.
- Every actor in Go-Akt:
    - has a process id [`PID`](./actors/pid.go). Via the process id any allowable action can be executed by the
      actor.
    - has a lifecycle via the following methods: [`PreStart`](./actors/actor.go), [`PostStop`](./actors/actor.go).
      It means it
      can live and die like any other process.
    - handles and responds to messages via the method [`Receive`](./actors/actor.go). While handling messages it
      can:
    - create other (child) actors via their process id [`PID`](./actors/pid.go) `SpawnChild` method
    - send messages to other actors locally or remotely via their process
      id [`PID`](./actors/pid.go) `Ask`, `RemoteAsk`(request/response
      fashion) and `Tell`, `RemoteTell`(fire-and-forget fashion) methods
    - stop (child) actors via their process id [`PID`](./actors/pid.go)
    - watch/unwatch (child) actors via their process id [`PID`](./actors/pid.go) `Watch` and `UnWatch` methods
    - supervise the failure behavior of (child) actors. The supervisory strategy to adopt is set during its
      creation:
    - Restart and Stop directive are supported at the moment.
    - remotely lookup for an actor on another node via their process id [`PID`](./actors/pid.go) `RemoteLookup`.
      This
      allows it to send messages remotely via `RemoteAsk` or `RemoteTell` methods
    - stash/unstash messages. See [Stashing](#stashing)
    - can adopt various form using the [Behavior](#behaviors) feature
    - can be restarted (respawned)
    - can be gracefully stopped (killed). Every message in the mailbox prior to stoppage will be processed within a
      configurable time period.

### Passivation

Actors can be passivated when they are idle after some period of time. Passivated actors are removed from the actor system to free-up resources.
When cluster mode is enabled, passivated actors are removed from the entire cluster. To bring back such actors to live, one needs to
`Spawn` them again. By default, all actors are passivated and the passivation time is `two minutes`.

- To enable passivation use the actor system option `WithExpireActorAfter(duration time.Duration)` when creating the actor system. See actor system [options](./actors/option.go).
- To disable passivation use the actor system option `WithPassivationDisabled` when creating the actor system. See actor system [options](./actors/option.go).

### Actor System

Without an actor system, it is not possible to create actors in Go-Akt. Only a single actor system
is recommended to be created per application when using Go-Akt. At the moment the single instance is not enforced in Go-Akt, this simple implementation is left to the discretion of the developer. To
create an actor system one just need to use
the [`NewActorSystem`](./actors/actor_system.go) method with the various [Options](./actors/option.go). Go-Akt
ActorSystem has the following characteristics:

- Actors lifecycle management (Spawn, Kill, ReSpawn)
- Concurrency and Parallelism - Multiple actors can be managed and execute their tasks independently and
  concurrently. This helps utilize multicore processors efficiently.
- Location Transparency - The physical location of actors is abstracted. Remote actors can be accessed via their
  address once _remoting_ is enabled.
- Fault Tolerance and Supervision - Set during the creation of the actor system.
- Actor Addressing - Every actor in the ActorSystem has an address.

### Behaviors

Actors in Go-Akt have the power to switch their behaviors at any point in time. When you change the actor behavior, the new
behavior will take effect for all subsequent messages until the behavior is changed again. The current message will
continue processing with the existing behavior. You can use [Stashing](#stashing) to reprocess the current
message with the new behavior.

To change the behavior, call the following methods on the [ReceiveContext interface](./actors/context.go) when handling a message:

- `Become` - switches the current behavior of the actor to a new behavior.
- `UnBecome` - resets the actor behavior to the default one which is the Actor.Receive method.
- `BecomeStacked` - sets a new behavior to the actor to the top of the behavior stack, while maintaining the previous ones.
- `UnBecomeStacked()` - sets the actor behavior to the previous behavior before `BecomeStacked()` was called. This only works with `BecomeStacked()`.

### Mailbox

Once can implement a custom mailbox. See [Mailbox](./actors/mailbox.go). The default mailbox makes use of buffered channels.

### Dead Letters

Dead letters are basically messages that cannot be delivered or that were not handled by a given actor.
Those messages are encapsulated in an event called [DeadletterEvent](./protos/goakt/events/v1/events.proto).
Dead letters are automatically emitted when a message cannot be delivered to actors' mailbox or when an Ask times out.
Also, one can emit dead letters from the receiving actor by using the `ctx.Unhandled()` method. This is useful instead of panicking when
the receiving actor does not know how to handle a particular message.
Dead letters are not propagated over the network, there are tied to the local actor system.
To receive the dead letter, you just need to call the actor system `Subscribe` and specify the `Event_DEAD_LETTER` event type.

### Messaging

Communication between actors is achieved exclusively through message passing. In Go-Akt _Google
Protocol Buffers_ is used to define messages.
The choice of protobuf is due to easy serialization over wire and strong schema definition. As stated previously the following messaging patterns are supported:

- `Tell/RemoteTell` - send a message to an actor and forget it
- `Ask/RemoteAsk` - send a message to an actor and expect a reply within a time period
- `Forward` - pass a message from one actor to the actor by preserving the initial sender of the message.
  At the moment you can only forward messages from the `ReceiveContext` when handling a message within an actor and this to a local actor.
- `BatchTell` - send a bulk of messages to actor in a fire-forget manner. Messages are processed one after the other in the other they have been sent.
- `BatchAsk` - send a bulk of messages to an actor and expect responses for each message sent within a time period. Messages are processed one after the other in the other they were sent.
  This help return the response of each message in the same order that message was sent. This method hinders performance drastically when the number of messages to sent is high.
  Kindly use this method with caution.

### Scheduler

You can schedule sending messages to actor that will be acted upon in the future. To achieve that you can use the following methods on the [Actor System](./actors/actor_system.go):

- `ScheduleOnce` - will send the given message to a local actor after a given interval
- `RemoteScheduleOnce` - will send the given message to a remote actor after a given interval. This requires remoting to be enabled on the actor system.
- `ScheduleWithCron` - will send the given message to a local actor using a [cron expression](#cron-expression-format).
- `RemoteScheduleWithCron` - will send the given message to a remote actor using a [cron expression](#cron-expression-format). This requires remoting to be enabled on the actor system.

#### Cron Expression Format

| Field        | Required | Allowed Values  | Allowed Special Characters |
|--------------|----------|-----------------|----------------------------|
| Seconds      | yes      | 0-59            | , - * /                    |
| Minutes      | yes      | 0-59            | , - * /                    |
| Hours        | yes      | 0-23            | , - * /                    |
| Day of month | yes      | 1-31            | , - * ? /                  |
| Month        | yes      | 1-12 or JAN-DEC | , - * /                    |
| Day of week  | yes      | 1-7 or SUN-SAT  | , - * ? /                  |
| Year         | no       | empty, 1970-    | , - * /                    |

#### Note

When running the actor system in a cluster only one instance of a given scheduled message will be running across the entire cluster.

### Stashing

Stashing is a mechanism you can enable in your actors, so they can temporarily stash away messages they cannot or should
not handle at the moment.
Another way to see it is that stashing allows you to keep processing messages you can handle while saving for later
messages you can't.
Stashing are handled by Go-Akt out of the actor instance just like the mailbox, so if the actor dies while processing a
message, all messages in the stash are processed.
This feature is usually used together with [Become/UnBecome](#behaviors), as they fit together very well, but this is
not a requirement.

It’s recommended to avoid stashing too many messages to avoid too much memory usage. If you try to stash more
messages than the capacity the actor will panic.
To use the stashing feature, call the following methods on the [ReceiveContext interface](./actors/context.go) when handling a message:

- `Stash()` - adds the current message to the stash buffer.
- `Unstash()` - unstashes the oldest message in the stash and prepends to the stash buffer.
- `UnstashAll()` - unstashes all messages from the stash buffer and prepends in the mailbox. Messages will be processed
  in the same order they arrived. The stash buffer will be empty after processing all messages, unless an exception is
  thrown or messages are stashed while unstashing.

### Remoting

This allows remote actors to communicate. The underlying technology is gRPC. To enable remoting just use the `WithRemoting` option when
creating the actor system. See actor system [options](./actors/option.go).

### Cluster

This offers simple scalability, partitioning (sharding), and re-balancing out-of-the-box. Go-Akt nodes are automatically discovered. See [Clustering](#clustering).

### Observability

Observability is key in distributed system. It helps to understand and track the performance of a system.
Go-Akt offers out of the box features that can help track, monitor and measure the performance of a Go-Akt based system.

#### Tracing

One can enable/disable tracing on a Go-Akt actor system to instrument and measure the performance of some of the methods.
Go-Akt uses under the hood [OpenTelemetry](https://opentelemetry.io/docs/instrumentation/go/) to instrument a system.
One just need to use the `WithTracing` option when instantiating a Go-Akt actor system and use the default [Telemetry](./telemetry/telemetry.go)
engine or set a custom one with `WithTelemetry` option of the actor system.

#### Logging

A simple logging interface to allow custom logger to be implemented instead of using the default logger.

### Testkit

To help implement unit tests in GoAkt-based applications. See [Testkit](./testkit)

## Use Cases

- Event-Driven programming
- Event Sourcing and CQRS - [eGo](https://github.com/Tochemey/ego)
- Highly Available, Fault-Tolerant Distributed Systems

## Installation

```bash
go get github.com/tochemey/goakt
```

## Clustering

The cluster engine depends upon the [discovery](./discovery/provider.go) mechanism to find other nodes in the cluster.
Under the hood, it leverages [Olric](https://github.com/buraksezer/olric)
to scale out and guarantee performant, reliable persistence, simple scalability, partitioning (sharding), and
re-balancing out-of-the-box.

At the moment the following providers are implemented:

- the [kubernetes](https://kubernetes.io/docs/home/) [api integration](./discovery/kubernetes) is fully functional
- the [mDNS](https://datatracker.ietf.org/doc/html/rfc6762) and [DNS-SD](https://tools.ietf.org/html/rfc6763)
- the [NATS](https://nats.io/) [integration](./discovery/nats) is fully functional
- the [DNS](./discovery/dnssd) is fully functional

Note: One can add additional discovery providers using the following [interface](./discovery/provider.go)

In addition, one needs to set the following environment variables irrespective of the discovery provider to help
identify the host node on which the cluster service is running:

- `NODE_NAME`: the node name. For instance in kubernetes one can just get it from the `metadata.name`
- `NODE_IP`: the node host address. For instance in kubernetes one can just get it from the `status.podIP`
- `GOSSIP_PORT`: the gossip protocol engine port.
- `CLUSTER_PORT`: the cluster port to help communicate with other GoAkt nodes in the cluster
- `REMOTING_PORT`: help remoting communication between actors

_Note: Depending upon the discovery provider implementation, the `GOSSIP_PORT` and `CLUSTER_PORT` can be the same.
The same applies to `NODE_NAME` and `NODE_IP`. This is up to the discretion of the implementation_

### Operations Guide

The following outlines the cluster mode operations which can help have a healthy GoAkt cluster:

- One can start a single node cluster or a multiple nodes cluster.
- One can add more nodes to the cluster which will automatically discover the cluster.
- One can remove nodes. However, to avoid losing data, one need to scale down the cluster to the minimum number of nodes
  which started the cluster.

### Built-in Discovery Providers

#### Kubernetes Discovery Provider Setup

To get the kubernetes discovery working as expected, the following pod labels need to be set:

- `app.kubernetes.io/part-of`: set this label with the actor system name
- `app.kubernetes.io/component`: set this label with the application name
- `app.kubernetes.io/name`: set this label with the application name

In addition, each node _is required to have three different ports open_ with the following ports name for the cluster
engine to work as expected:

- `gossip-port`: help the gossip protocol engine. This is actually the kubernetes discovery port
- `cluster-port`: help the cluster engine to communicate with other GoAkt nodes in the cluster
- `remoting-port`: help for remoting messaging between actors

##### Get Started

```go
const (
    namespace = "default"
    applicationName = "accounts"
    actorSystemName    = "AccountsSystem"
)
// instantiate the k8 discovery provider
disco := kubernetes.NewDiscovery()
// define the discovery options
discoOptions := discovery.Config{
    kubernetes.ApplicationName: applicationName,
    kubernetes.ActorSystemName: actorSystemName,
    kubernetes.Namespace:       namespace,
}
// define the service discovery
serviceDiscovery := discovery.NewServiceDiscovery(disco, discoOptions)
// pass the service discovery when enabling cluster mode in the actor system
```

##### Role Based Access

You’ll also have to grant the Service Account that your pods run under access to list pods. The following configuration
can be used as a starting point.
It creates a Role, pod-reader, which grants access to query pod information. It then binds the default Service Account
to the Role by creating a RoleBinding.
Adjust as necessary:

```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pod-reader
rules:
  - apiGroups: [""] # "" indicates the core API group
    resources: ["pods"]
    verbs: ["get", "watch", "list"]
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-pods
subjects:
  # Uses the default service account. Consider creating a new one.
  - kind: ServiceAccount
    name: default
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

##### Sample Project

A working example can be found [here](./examples/actor-cluster/k8s) with a
small [doc](./examples/actor-cluster/k8s/doc.md) showing how to run it.

#### mDNS Discovery Provider Setup

- `Service Name`: the service name
- `Domain`: The mDNS discovery domain
- `Port`: The mDNS discovery port
- `IPv6`: States whether to lookup for IPv6 addresses.

#### NATS Discovery Provider Setup

To use the NATS discovery provider one needs to provide the following:

- `NATS Server Address`: the NATS Server address
- `NATS Subject`: the NATS subject to use
- `Actor System Name`: the actor system name
- `Application Name`: the application name

```go
const (
    natsServerAddr = "nats://localhost:4248"
    natsSubject = "goakt-gossip"
    applicationName = "accounts"
    actorSystemName = "AccountsSystem"
)
// instantiate the NATS discovery provider
disco := nats.NewDiscovery()
// define the discovery options
discoOptions := discovery.Config{
    ApplicationName: applicationName,
    ActorSystemName: actorSystemName,
    NatsServer:      natsServer,
    NatsSubject:     natsSubject,
}
// define the service discovery
serviceDiscovery := discovery.NewServiceDiscovery(disco, discoOptions)
// pass the service discovery when enabling cluster mode in the actor system
```

#### DNS Provider Setup

This provider performs nodes discovery based upon the domain name provided. This is very useful when doing local development
using docker.

To use the DNS discovery provider one needs to provide the following:

- `Domain Name`: the NATS Server address
- `IPv6`: States whether to lookup for IPv6 addresses.

```go
const domainName = "accounts"
// instantiate the dnssd discovery provider
disco := dnssd.NewDiscovery()
// define the discovery options
discoOptions := discovery.Config{
    dnssd.DomainName: domainName,
    dnssd.IPv6:       false,
}
// define the service discovery
serviceDiscovery := discovery.NewServiceDiscovery(disco, discoOptions
// pass the service discovery when enabling cluster mode in the actor system
```

##### Sample Project

There is an example [here](./examples/actor-cluster/dnssd) that shows how to use it.

## Examples

Kindly check out the [examples'](./examples) folder.

## Contribution

Contributions are welcome!
The project adheres to [Semantic Versioning](https://semver.org)
and [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).
This repo uses [Earthly](https://earthly.dev/get-earthly).

To contribute please:

- Fork the repository
- Create a feature branch
- Submit a [pull request](https://help.github.com/articles/using-pull-requests)

### Test & Linter

Prior to submitting a [pull request](https://help.github.com/articles/using-pull-requests), please run:

```bash
earthly +test
```

## Benchmark

One can run the benchmark test: `go test -bench=. -benchtime 2s -count 5 -benchmem -cpu 8 -run notest` from
the [bench package](./bench) or just run the command `make bench`.
