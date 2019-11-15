<div align="center">
					<object type="image/png" data="wasm/assets/images/baby-groot.svg" width="250">
						<img src="https://github.com/will-rowe/baby-groot/raw/master/misc/baby-groot-logo.png" alt="logo" width="250">
					</object>
    <h3>BABY GROOT & THE WASP</h3>
</div>

***

## Overview

### BABY-GROOT

BABY-GROOT is a dummer, smaller version of [GROOT](https://github.com/will-rowe/groot).

The main difference between BABY-GROOT and GROOT is that we no longer use exact alignments to detect variants. Instead, the seeded reads are used to create `approximately weighted variation graphs`. [Expectation Maximization](https://www.statisticshowto.datasciencecentral.com/em-algorithm-expectation-maximization/) is then used to predict the most likely graph traversals based on the node weightings.

Another key difference is that the variation graphs are now indexed using [LSH Ensemble](http://www.vldb.org/pvldb/vol9/p1185-zhu.pdf) instead of LSH Forest, which allows for containment search. This means that BABY-GROOT supports variable read length search queries.

These 2 changes have improved the efficiency of GROOT, such that we can now port it to WebAssembly (WASM) and have it run in a browser.

### The WASP

The WASP part of this repo is a [WebAssembly](https://developer.mozilla.org/en-US/docs/WebAssembly) port of the BABY-GROOT command line tool, so that you can run it via a web app. This is currently hosted [here](https://willrowe.net/baby-groot). The cool thing about WASP (and web assembly in general) is that all the work is done client side - so no files leave your machine and you aren't reliant on a server.

***

## Weighting variation graphs

The basic idea behind BABY GROOT is:

* use [sketches](https://github.com/will-rowe/genome-sketching) + an [LSH Ensemble index](http://www.vldb.org/pvldb/vol9/p1185-zhu.pdf) to approximately map reads to variation graphs
* use these mappings to create approximately weighted variation graphs (in GFA format)
  * we know a read approximately maps between nodes X and Z on a graph (with approx. error)
  * we used k-mer decomposition to generate a sketch
  * so nodes X through Z will have A k-mers, where A = length(X..Z) - k + 1 
  * we can distribute A k-mers across nodes X, Y and Z
* use EM to probabilistically determine the best paths through the weighted graphs
  * the best paths are equivalent to the gene variants in our sample
* simplify the graph windowing
  * windows which have identical sketches are merged
  * instead of subpaths, have contained nodes (i.e. a sketch represents a set of nodes in the graph, which can come from multiple traversals)
  * no longer need offsets etc. - just weighting all nodes, regardless of sketch offset on a segment

***

## Changes and new features since the GROOT publication

The original [GROOT paper](https://academic.oup.com/bioinformatics/article/34/21/3601/4995843) was published in Bioinformatics last year. BABY GROOT has seen a few changes:

* replaced the indexing scheme (LSH Forest -> LSH Ensemble)
* replaced existing hash functions
  * now only consider canonical k-mers
* removed exact alignment step and BAM outputs
* replaced align subcommand with sketch
* replaced report subcommand with haplotype
* output weighted GFA as the primary output
* general code improvements and bug fixes
* added support for [Go modules](https://github.com/golang/go/wiki/Modules) and Go 1.13
* added a WASM port
* added a basic API so components can be more easily called from other Go programs

***

## TODO

* swap serialisation to messagepack (or maybe protobuf) from gob
* improve WASM
  * currently only small index file sizes are supported before browser complains
  * add in more control over runtime parameters for BABY-GROOT
