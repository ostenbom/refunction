FROM alpine:3.8

RUN apk add --no-cache gcc musl-dev
WORKDIR /sleep-work
COPY sleep.c .
RUN gcc /sleep-work/sleep.c -o /sleep-work/sigusr-sleep

FROM alpine:3.8
COPY --from=0 /sleep-work/sigusr-sleep /bin/sigusr-sleep
CMD ["/bin/sigusr-sleep"]
