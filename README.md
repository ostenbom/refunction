# refunction
[![Build Status](https://travis-ci.com/ostenbom/refunction.svg?branch=master)](https://travis-ci.com/ostenbom/refunction)

The next generation of cloud hosting is known as serverless or function-as-a-service (Faas). The popularity of these easily deployable and infinitely scalable functions has skyrocketed since AWS announced their Lambda service in 2014. One primary concern regarding the performance of these functions is the effect of “cold starts”. A cold start is the time it takes to boot a new container when the platform needs to increase its capacity for that function.

We investigate the possibility of restoring function containers as an alternative to starting new containers.

Our method focuses on Linux process primitives. We store and modify state such as raw memory and registers in order to reset the process to the way it was before the user’s function was loaded. We discuss how to ensure temporal isolation in order to provide security guarantees in such a system. We find that it is possible to restore container processes in a variety of runtimes. Using this approach can decrease the effect of cold starts by up to 20x and increase the overall throughput of such systems. 

[The full report can be found here](report.pdf)
