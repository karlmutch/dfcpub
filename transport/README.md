## Overview

Package transport provides streaming object-based transport over HTTP for massive intra-DFC and DFC-to-DFC data transfers. This transport layer
can be utilized for cluster-wide rebalancing, replication of any kind, distributed merge-sort operations, and more.

First, basic definitions:

| Term | Description | Example |
|--- | --- | ---|
| Stream | A point-to-point flow over one or multiple HTTP PUT requests (and as many TCP connections) | `transport.NewStream(client, "http://example.com")` - creates a stream between the local client and the `example.com` host |
| Object | Any sequence of bytes (or, more precisely, any [io.ReadCloser](https://golang.org/pkg/io/#ReadCloser)) that is accompanied by a transport header | `transport.Header{"abc", "X", nil, 1024*1024}` - specifies a 1MB object that will be named `abc/X` at the destination |
| Object Header | A `transport.Header` structure that, in addition to bucket name, object name, and object size, carries an arbitrary (*opaque*) sequence of bytes that, for instance, may be a JSON message or anything else. | `transport.Header{"abracadabra", "p/q/s", []byte{'1', '2', '3'}, 13}` - describes a 13-byte object that, in the example, has some application-specific and non-nil *opaque* field in the header |
| Receive callback | A function that has the following signature: `Receive func(http.ResponseWriter, transport.Header, io.Reader)`. Receive callback must be *registered* prior to the very first object being transferred over the stream - see next. | Notice the last parameter in the receive callback: `io.Reader`. Behind this (reading) interface, there's a special type reader supporting, in part, object boundaries. In other words, each callback invocation corresponds to one ransferred and received object. Note as well the object header that is also delivered to the receiving endpoint via the same callback. |
| Registering receive callback | An API to establish the one-to-one correspondence between the stream sender and the stream receiver | For instance, to register the same receive callback `foo` with two different HTTP endpoints named "ep1" and "ep2", we could call `transport.Register("n1", "ep1", foo)` and `transport.Register("n1", "ep2", foo)`, where `n1` is an http request multiplexer ("muxer") that corresponds to one of the documented networking options - see [README, section Networking](README.md). The transport will then be calling `foo()` to separately deliver the "ep1" stream to the "ep1" endpoint and "ep2" - to, respectively, "ep2". Needless to say that a per-endpoint callback is also supported and permitted. To allow registering endpoints to different http request multiplexers, one can change network parameter `transport.Register("different-network", "ep1", foo)` |

## Example with comments

```go
path := transport.Register("n1", "ep1", testReceive) // register receive callback with HTTP endpoint "ep1" to "n1" network
client := &http.Client{Transport: &http.Transport{}} // create default HTTP client
url := "http://example.com/" +  path // combine the hostname with the result of the Register() above
stream := transport.NewStream(client, url) // open a stream to the http endpoint identified by the url

for  {
	hdr := transport.Header{...} 	// next object header
	object := ... 			// next object reader, e.g. os.Open("some file")
	stream.SendAsync(hdr, object)	// send the object asynchronously
	...
}
stream.Fin() // gracefully close the stream

```
## Registering HTTP endpoint

On the receiving side, each network contains multiple HTTP endpoints, whereby each HTTP endpoint, in turn, may have zero or more stream sessions.
In effect, there are two nested many-to-many relationships whereby you may have multiple logical networks, each containing multiple named transports, etc.

The following:

```go
path, err := transport.Register("public", "myapp", mycallback)
```

adds a transport endpoint named "myapp" to the "public" network (that must already exist), and then registers a user callback with the latter.

The last argument, user-defined callback, must have the following typedef:

```go
Receive func(w http.ResponseWriter, hdr Header, object io.Reader)
```

The callback is being invoked on a per received object basis (note that a single stream may transfer multiple, potentially unlimited, number of objects).

Back to the registration. On the HTTP receiving side, the call to `Register` translates as:

```go
mux.HandleFunc(path, mycallback)
```
where mux is `http.ServeMux` that corresponds to the named network ("public", in this example), and path is a URL path ending with "/myapp".

**BEWARE**

>> HTTP request multiplexer matches the URL of each incoming request against a list of registered paths and calls the handler for the path that most closely matches the URL.

>> That is why registering a new endpoint with a given network (and its per-network multiplexer) should not be done concurrently with traffic that utilizes this same network.

>> The limitation is rooted in the fact that, when registering, we insert an entry into the `http.ServeMux` private map of all its URL paths. This map is protected by a private mutex and is read-accessed to route HTTP requests...


## On the wire

On the wire, each transmitted object will have the layout:

>> [header length] [header fields including object name and size] [object bytes]

The size must be known upfront, which is the current limitation.

A stream (the [Stream type](transport/send.go)) carries a sequence of objects of arbitrary sizes and contents, and overall looks as follows:

>> object1 = (**[header1]**, **[data1]**) object2 = (**[header2]**, **[data2]**), etc.

Stream termination is denoted by a special marker in the data-size field of the header:

>> header = [object size=7fffffffffffffff]

## Testing

* To run all tests while redirecting errors to standard error:
```
go test -v -logtostderr=true
```

* To run a given test (with a name matching "Multi") and enabled debugging:
```
DFC_STREAM_DEBUG=1 go test -v -run=Multi
```