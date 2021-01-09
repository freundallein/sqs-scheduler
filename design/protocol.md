## Scheduler protocol  

Use JSON-RPC 2.0 as transport-independent protocol.

For example, we use scheduler as data export tool&

First of all, we need to submit an export task to InboundQueue:
### Submit task
```
{"jsonrpc": "2.0", "method": "submit:export", "params": {"objectID": 23}}
```

Then scheduler should send it to OutboundQueue:

### Enqueue task
```
{"jsonrpc": "2.0", "method": "export", "params": {"objectID": 23}, "id": 1}
```

And we want to receive results via ResultsQueue:

### Return result
```
{"jsonrpc": "2.0", "result": {"result": "success", "attempt": 1}, "id": 1}
```
or
```
{"jsonrpc": "2.0", "error": {"code": -1234, "message": "something bad happened", "attempt": 1}, "id": 1}
```