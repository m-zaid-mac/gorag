# Go Concurrency Basics

Goroutines are lightweight threads managed by the Go runtime rather than
the operating system. Starting one costs only a few kilobytes of stack,
so it's common to spin up thousands of them, unlike OS threads which are
much more expensive to create and switch between.

Channels are the primary way goroutines communicate and synchronize.
A channel is typed, so only values of that type can be sent through it.
Unbuffered channels block the sender until a receiver is ready, which
makes them useful for handshake-style synchronization. Buffered channels
allow a fixed number of values to be queued before the sender blocks,
which is useful for worker pools where you want to decouple producers
from consumers up to a point.

A worker pool is a common pattern: a fixed number of goroutines read
from a shared jobs channel, process each job, and send results to a
separate results channel. This bounds concurrency so you don't overwhelm
a downstream resource like a rate-limited API, while still processing
many jobs in parallel instead of one at a time.

The `context` package carries deadlines, cancellation signals, and
request-scoped values across API boundaries and between goroutines. It's
the idiomatic way to say "stop what you're doing" to a tree of goroutines,
for example when an HTTP request is cancelled by the client.
