<div align="center">
					<object type="image/png" data="wasm/assets/css/images/baby-groot.svg" width="250">
						<img src="https://github.com/will-rowe/baby-groot/raw/master/baby-groot-logo.png" alt="logo" width="250">
					</object>
    <h3>BABY GROOT & THE WASP</h3>
</div>

***

## Overview

### BABY GROOT

BABY GROOT is a dummer, smaller version of [GROOT](https://github.com/will-rowe/groot). This repo is essentially for the development of GROOT v2.0 and for trying out some new ideas in my spare time.

The main difference between BABY GROOT and GROOT is that we no longer use exact alignments to detect variants. Instead, the seeded reads are used to create `approximately weighted variation graphs`. [Expectation Maximization](https://www.statisticshowto.datasciencecentral.com/em-algorithm-expectation-maximization/) is then used to predict the most likely graph traversals based on the node weightings.

The hope is that this new method will be less sensitive to sequencing error and allow variable read lengths (including long reads), whilst still performing as well as GROOT v1.0.

Hopefully this project will transition into a more generalised tool, which can be used for efficient haplotype recovery.


### WASP

The WASP part of this repo is a [web assembly](https://developer.mozilla.org/en-US/docs/WebAssembly) port of the BABY GROOT command line tool, so that you can run it via a Graphical User Interface (GUI). This is currently hosted [here](https://will-rowe.github.io/baby-groot/). The cool thing about WASP (and web assembly in general) is that all the work is done client side - so no files leave your machine and you aren't reliant on a server.

WASP is still a work in progress and currently is only running on small files in my test set up.

***

## Weighting variation graphs

The basic idea behind BABY GROOT is:

* use [data sketching](https://github.com/will-rowe/genome-sketching) + an [LSH forest index](http://infolab.stanford.edu/~bawa/Pub/similarity.pdf) to approximately map reads to variation graphs
* use these mappings to create approximately weighted variation graphs (in GFA format)
  * we know a read approximately maps between nodes X and Z on a graph (with approx. error)
  * we used k-mer decomposition to generate a sketch
  * so nodes X through Z will have A k-mers, where A = length(X..Z) - k + 1 
  * we can distribute A k-mers across nodes X, Y and Z
* use EM to probabilistically determine the best paths through the weighted graphs
  * the best paths are equivalent to the gene variants in our sample

***

## Changes and new features since the GROOT publication

The original [GROOT paper](https://academic.oup.com/bioinformatics/article/34/21/3601/4995843) was published in Bioinformatics last year. BABY GROOT has seen a few changes:

* replaced existing hash functions with ntHash
  * also, now only consider canonical k-mers
* added extra sketching algorithms for mapping reads to graph traversals
  * added bottom-k minhash algorithm, in addition to existing KMV minhash
  * added bloom filter to stop unique k-mers being added to sketches
* removed exact alignment step and BAM outputs
* replaced align subcommand with sketch
* replaced report subcommand with haplotype
* output weighted GFA as the primary output
* swapped serialisation to messagepack (json derivative) from gob
* general code improvements and bug fixes
* added support for [Go modules](https://github.com/golang/go/wiki/Modules) and Go 1.12
* added a WASM port
* added a basic API so components can be more easily called from other Go programs

***

## Still to come

* additional support for variable read lengths and long reads
  * I'm thinking that some sort of windowing strategy of the query sequences may work out better than containment, now that we're not doing exact alignment
* additional sketching algorithms to benchmark against existing KMV, Bottom-K, Bloom Filter
* add in phasing and paired-end information for informing path prediction during haplotype recovery
* improving node re-weighting scheme during MCMC path finding
* add in the option to augment graphs with new paths during mapping
* test out MCMC as alternative to EM
* improve WASM
  * currently only small file sizes are supported before browser complains
  * need to improve javascript data streaming function
  * add in more control over runtime parameters for BABY-GROOT