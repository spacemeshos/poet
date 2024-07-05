# PoET Operator Manual

Proof of Elapsed Time, or PoET, is a cryptographic primitive used to prove that a specific amount of real (clock) time has elapsed.

You can think about PoET like an HTTP web service with a twist.

## PoET important endpoints

* `https://POET_URL/v1/pow_params`
    nodes will query that URL to get the POW challenge parameters.
    it will change every round, safe to cache in all other moments.
* `https://POET_URL/v1/info`
    contains the current round id and the current round state. Nodes currently don't actively use that endpoint besides querying and checking if poet is alive. Safe to cache and invalidate when round state changes.
* `https://POET_URL/v1/submit`
    nodes will submit their PoET proofs to that endpoint. *Never cache*. The POW serves as a rate-limiting factor and is meant to protect PoET from DOS.
    The higher the value of `pow-difficulty`, the bigger the protection but also more burden on the nodes.
* `https://POET_UR/v1/proofs/{round_id}`
    nodes will query that endpoint to get the PoET proofs for a given round. Once data is there then valid forever. Safe to cache.

PoET exposes the above API as:
- REST endpoints via the [GRPC gateway](https://github.com/grpc-ecosystem/grpc-gateway)
- GRPC

The reference node implementation [go-spacemesh](https://github.com/spacemeshos/go-spacemesh) expects the REST API so it's recommended to expose it publicly. Exposing GRPC is not required.

> [!NOTE]
> We're working to expose the cache time for all the endpoints to safely cache that on the cache servers https://github.com/spacemeshos/poet/issues/330

## PoET Overview
Poet round have two states `open` `in-progress`.
* `open` others can submit to the given round
* `in-progress` no more submissions are allowed AND PoET builds PoSW (a merkle tree)

When the round is open poet is pretty much a web application with people using `info` `pow_params` `submit` endpoints. There is a metric `grpc_server_handled_total` that will give the exact numbers etc more info in (#metrics).

### Config files

Please see [poet.service](./poet.service) and [poet_config_sample.conf](./poet_config_sample.conf).


### Challenge registration
Users submit their "challenge tokens" for the open round to participate in the following round PoSW.
There are two ways that submits can be verified

#### 1. PoW
The submitter can solve a proof of work and pass the found nonce in the call to /submit.
The PoW difficulty is configurable via `pow-difficulty` config param. The reasonable values are within 20-26.
The difficulty value means the number of leading zero bits in the hash that the solver must find.

Note: PoW is deprecated in favor of certification (see below)

#### 2. Certification
The submitter can certify within a service that poets recognizes (and points to in /info) and pass the certificate in /submit call. The poet will then verify if the certificate is valid and allow the registration if it is.
A certificate is the node's public key signed by the certifier service.

To configure the certifier, pass its URL and public key (base64-encoded) in the config:
```toml
certifier-url="http://certifier.service"
certifier-pubkey="Zm9vYmFy"
```

If the certifier is not configured, it will be disabled and poet will not accept registrations
using certificates - it will fallback to PoW.

##### Canonical certifier service setup
The certifier service implemented by Spacemesh is available at https://github.com/spacemeshos/post-rs/tree/v0.6.0/certifier. Follow its README to deploy your own certifier service.

## PoET Round States and the tree

When the poet round is `in-progress` it generates the PoSW - proof of sequential work (essentially merkle tree). It will write the tree to a disk to a directory specified by `--datadir` flag. The per round tree will be saved in that directory in a subdirectory named after the round id (`<datadir>/rounds/<round-id>`).

```.
<datadir>
└── rounds
    └── 3
        ├── layercache_0.bin
        ├── layercache_1.bin
        ...
        └── state.bin
```

That directory listing is from a poet that was having round `3` in `in-progress` state, because as you can see there are layercache files. Each of these files is a layer (level) of the PoSW tree.

Each of these files should be half of the size of the previous one (`0` being biggest).


After round is finished poet will delete the tree from the disk and start a new round after `cycle_gap` ends.

### Layercache

PoET tries to optimize the cache usage by knowing how much `lps` (leafs per second) is it expected to get. You can specify that with `--lps` flag. To obtain the value the best is to use `poet_bench` tool but please run it for significant amount of time (at least 10 minutes) to get the stable results.

By default poet uses 26 memory layers ( Up to (1 << 26) * 2 - 1 Merkle tree cache nodes (32 bytes each) will be held in-memory), the other layers are stored on disk. If your poet generates 10_000 leafs per second then the expected disk storage for a 13,5days running would be `10_000 * 60 * 60 * 24 * 13,5 = 11_664_000_000` leafs. That means that the produced tree will have `11_664_000_000`. But to produce `11_664_000_000` leafs you need `11_664_000_000 - 1` nodes above them. So we can multiply `11_664_000_000` by 2 to get the total tree size which is: `23_328_000_000`. If you multiply it by `32bytes` (size of one node) then you get `373.248 * 2 GB` which is `746.496 GB` for a PoET service that runs for 13,5 days and generates 10_000 leafs per second. That's an approximation because the tree is not always full. But it's a good enough approximation. Some of the layers (`--memory` flag and `MemoryLayers` config option) are in the memory so that should serve as the upper bound.

Layers are written to a disk with buffered file, you can configure the buffer size by using `tree-file-buffer`, the default is `4096` and should be good enough for most cases.

You can also use that logic to estimate needed disk performance and IOps. But HDD without a SSD cache should be more than sufficient in any case.

You *must* make sure that the data is persisted. The existence of the tree is essential to generate proof for a given round. If the tree is not there then it's not possible to generate the proof. The proof is a Merkle Proof proving selected `T` leaves from the tree and basically contains selected nodes from the tree.

> [!NOTE]
> PoET will *not* start generating tree when it will detect that the round should be already in progress but the state of the round is empty. This is to prevent the situation when PoET is doing work for nothing

## Performance fine tunning

Because PoET PoSW generation is single threaded and CPU bound, it is important to fine tune the CPU and scheduler to get the best performance. On Linux platform we strongly recommend using kernel 6.2 and newer.

When the poet round is starting poet process will create `round.tid` located in the poet data directory. This file contains the thread id of the poet process that builds the tree. You can use this thread id to fine tune the CPU and scheduler. You can use `taskset` and `chrt` to set the CPU affinity and scheduler priority. That process id will not change for the entire round (unless poet service is restarted).

In general for given CPU you need to make it clock as high as possible and sustain it for long period of time.

### Intel

Currently the best known Intel processor is i9-13900KS. Depending on the silicon lottery you might need to disable all but one fastest `P` core in BIOS (best option) or you could use the following:
`echo 0 > /sys/devices/system/cpu/cpu${i}/online` where `i` is the core number. This will disable the core. You can check the core number with `lscpu` command.

### AMD

The best AMD processor for PoET is 7950x. You might need to disable the slower CCD and then use PBO and other overclocking options to get the best performance for single core.

### HDD

For `ext4` filesystem you might want to experiment with `noatime,data=writeback` `noatime` is obvious win, `data=writeback` is a bit more tricky and needs to be considered, because in sudden power loss or crash you may endup with inconsistent FS state.
With 10_000 lps (leafs per second) you will get ABSOLUTELY at most 20_000 * 32bytes write per second (0.64MB).
Playing with the buffer size `tree-file-buffer` might be a good idea also.


# Monitoring & Maintenace

## Metrics

PoET exposes metrics with standard prometheus compatible endpoint. You can configure the port with `--metrics-port` flag. The metrics are exposed on `/metrics` endpoint. PoET is golang application so all `go_*` metrics are also exposed by default.

* `poet_round_leaves_total` - total number of lps per round, you can use `rate(poet_round_leaves_total{}[$__rate_interval])  != 0` prometheus query to do rate based metrics, sudden drop might indicate CPU problems or POET issue.
* `poet_round_members_total` - total number of members per round
* `grpc_server_handled_total` - contains metrics about the GRPC server endpoints, use it to monitor the specific behaviour in the PoET service
    * When the round is open make sure that `Submit` and `Proof` succeeds
    * Make sure that `Info` always is `Ok`
    * When there are requestst to `Proof` sometimes the proof on the poet side might be not yet ready then clients will retry. That's normal and expected behaviour.
* `poet_round_batch_write_*` - starting with `0.9.1` we started to use batches to write members to a disk, that metric would be helpfull to monitor for bottlenecks during the open round.
* `go_*` monitor like normal go application.
