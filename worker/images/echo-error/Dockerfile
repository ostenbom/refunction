FROM alpine:3.8

RUN apk add --no-cache gcc musl-dev
WORKDIR /work
COPY echo-hello.c .
RUN gcc /work/echo-hello.c -o /work/echo-hello

FROM alpine:3.8
COPY --from=0 /work/echo-hello /bin/echo-hello
CMD ["echo-hello"]
