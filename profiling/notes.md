# Notes on current mem and cpu profiling

## Indexing

Run using ARG-annot db as input

High-mem and cpu cycles for windowing function are likely due to the fact that each graph is processed concurrently. Need to restrict the number of go routines here.

## Sketching

Run using a E.coli genome with 3M reads

Swapping to Protobuf3 from msgpack has resulted in 75% reduction in memory usage during index load

The number of heap allocated objects are rising steadily to occupy to 2Gb after 1M reads (up from 1Gb after initial 100,000 reads) - need to check why so many objects are persisting in the heap. Either there is a bottleneck somewhere or objects aren't being collected
