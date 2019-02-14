```
           __                  _   _             
 _ __ ___ / _|_   _ _ __   ___| |_(_) ___  _ __  
|  __/ _ \ |_| | | | '_ \ / __| __| |/ _ \| '_ \ 
| | | \__/  _| |_| | | | | (__| |_| | (_) | | | |
|_|  \___|_|  \__,_|_| |_|\___|\__|_|\___/|_| |_|
```

# refunction
[![Build Status](https://travis-ci.com/ostenbom/refunction.svg?branch=master)](https://travis-ci.com/ostenbom/refunction)

Reusing containers for faster serverless function execution - Masters Project @ Imperial College

The latest fad in lightweight virtualisation primitives is known as serverless.  Rather than renting a portion of a server via a container or VM on a monthly basis, serverless platforms promise billing by the 1/10th of a second and infinite scaling exactly according to demand with almost no configuration. These features promise reduced developer operations, a lower barrier to entry for the cloud and reduced running costs.

Currently, serverless containers can experience high latencies on startup, known as a “cold start”.

This project aims to explore the possibility of reusing containers when running serverless functions, such that different users can use the same container, one after another, with a cleanup step in between.
