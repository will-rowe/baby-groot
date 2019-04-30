# Recovering a blaOXA allele

This example shows how we can recover a blaOXA allele - blaOXA-90.

This allele clusters with 81 other variants of blaOXA (90% clustering identity).

***

* simulate the reads with BBMap

```
randomreads.sh ref=blaOXA-90.fna out=blaOXA-90-reads.1000.fq length=100 reads=1000
```

* create and index the graphs

```
go run  ../../main.go index -i arg-annot.90 -o test-index -k 31 -s 82 -j 0.99 -p 4
```

* align the reads

```
go run ../../main.go align -i test-index -f blaOXA-90-reads.1000.fq -o test-graphs -p 4
```

* predict the haplotype

```
go run ../../main.go haplotype -i test-index -g test-graphs -o test-haplotype -b 100 -z 0.97 -s 0.005 -p 4
```
