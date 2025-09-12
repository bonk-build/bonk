# bonk the builder
*Your favorite build system's favorite build system.*

---

Bonk is a build system _framework_, designed to be an extensible and unopinionated task executor.
Bonk itself is relatively simple: it only consists of the task & executor definitions, as well as a few useful executors.
A more complete bonk system is comprised of the bonk task scheduler and a series of linked executors.

Conceptually, bonk could replace anything and everything from GNU make to package managers.

### How it works

Executing a bonk build takes the following steps:

1. a bonk frontend loads any required (un-linked) plugins.
1. the frontend spawns 1 or more tasks to process the source tree and emit followup tasks.
1. the followup tasks are scheduled according to dependencies to fully build/process the source tree.
1. ???
1. profit

### Executors

This document has mentioned executors several times so far, but what is an executor?
An executor is simply an object that conforms to the [Executor interface](https://go.bonk.build/pkg/task#Executor) to process tasks sent to it.
Where this gets interesting is with the idea of _chaining_ executors.
Because executors may wrap eachother to perform pre/post-processing and forwarding, the possibilities of where/how executors run is limitless*.

The bonk built-in executors include:

1. [tree](https://go.bonk.build/pkg/executor/tree): forwards tasks to the appropriate child by name.
1. [rpc](https://go.bonk.build/pkg/executor/rpc): forwards all tasks sent to it across a gRPC connection.
1. [plugin](https://go.bonk.build/pkg/executor/plugin): uses [go-plugin](https://github.com/hashicorp/go-plugin) to spawn a subprocess and open an rpc connection.

### Frontends

The frontend is the "driver" of bonk. This tool is responsible for creating the required bonk infrastructure and spawning the initial task(s).

This may be as simple as the built-in bonk frontend (which parses all of this information from a config file), or more opinionated for working with more established workflows.

### How to use bonk

You can't yet, I'm still working on it.
