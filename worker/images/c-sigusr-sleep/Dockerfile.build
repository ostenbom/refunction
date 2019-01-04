FROM alpine:3.8

RUN apk add --no-cache gcc musl-dev
COPY sleep.c /bin/
RUN gcc /bin/sleep.c -o /bin/sigusr-sleep

CMD ["/bin/sigusr-sleep"]
