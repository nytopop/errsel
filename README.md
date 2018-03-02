# errsel
![](https://forthebadge.com/images/badges/powered-by-electricity.svg)

[![](https://godoc.org/github.com/nytopop/errsel?status.svg)](http://godoc.org/github.com/nytopop/errsel)
[![Build Status](https://travis-ci.org/nytopop/errsel.svg?branch=master)](https://travis-ci.org/nytopop/errsel)

Errsel is an experiment in writing functional Go, and how it performs in comparison with imperative logic. Interestingly enough, the imperative versions of what i tested aren't much faster at execution time (+/- 5%). There is significant (~50%) overhead when constructing functional computations, but due to their immutability that only needs to be done once - they can just be reused indefinitely after the initial allocation.
