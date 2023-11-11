# Piano: Extremely Simple, Single-server PIR with Sublinear Server Computation

This is a prototype implementation of the Piano private information retrieval(PIR) algorithm that allows a client to access a database without the server knowing the querying index.
The algorithm details can be found in the paper (https://eprint.iacr.org/2023/452.pdf).

**Note**: 
This repo includes the both version of Piano (the initial version and the updated version). 
The new version is now described in the main body of the online version of our paper.

Th initial version is now described as a variant in the online version of our paper (Appendix A).
The legacy code is in `client/client.go`.
Running the code will get the numbers we report in our conference version (https://www.computer.org/csdl/proceedings-article/sp/2024/313000a055/1RjEaufvKzm).

**Warning**: The code is not audited and not for any serious commercial or real-world use case. Please use it only for educational purpose.

### Prerequisite:
1. Install Go(https://go.dev/doc/install).
2. For developing, please install gRPC(https://grpc.io/docs/languages/go/quickstart/)

### A Mini Tutorial

The tutorial implementation is in `tutorial_new/tutorial_new.go`.

Try `go run tutorial_new/tutorial_new.go`.

### Running Experiments:
1. In one terminal, `go run server/server.go -port 50051`. This sets up the server. The server will store the whole DB in the RAM, so please ensure there's enough memory.
2. In another terminal, `go run client_new/client_new.go -ip localhost:50051 -thread 1`. This runs the PIR experiment with one setup phase for a window of $\sqrt{n}\ln(n)$-queries and follows with the online phase of up to 1000 queries. The ip flag denotes the server's adddress. The thread denotes how many threads are used in the setup phase.

#### Different DB configuration:
1. The two integers in `config.txt` denote `N` and `DBSeed`. `N` denotes the number of entries in the database. `DBSeed` denotes the random seed to generate the DB. The client will use the seed only for verifying the correctness. The code only reads the integers in the first line.
2. In `util/util.go`, you can change the `DBEntrySize` constant to change the entry size, e.g. 8bytes, 32bytes, 256bytes.

For example, setting `N=134217728`  and `DBEntrySize=8` will generate an 1GB database.

### Developing
1. The server implementation is in `server/server.go`.
2. The client implementation is in `client/client.go`.
3. Common utilities are in `util/util.go`, including the PRF and the `DBEntry` definition.
4. The messages exchanged by servers and client are defined in `query/query.proto`. If you change it, run `bash proto.sh` to generate the corresponding server and client API. You should implement those API later.

### Contact
Mingxun Zhou(mingxunz@andrew.cmu.edu)

Andrew Park(ahp2@andrew.cmu.edu)
