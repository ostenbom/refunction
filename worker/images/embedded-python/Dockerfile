FROM python:3.6.8-alpine3.8

RUN apk add --no-cache gcc musl-dev python3-dev
WORKDIR /embed-work
COPY *.c *.h ./
# RUN /usr/bin/python3.6-config --cflags
# RUN /usr/bin/python3.6-config --ldflags
RUN gcc -c embedded-python.c cJSON.c -I/usr/include/python3.6m -I/usr/include/python3.6m  -Wno-unused-result -Wsign-compare -Os -fomit-frame-pointer -g -DNDEBUG -g -fwrapv -O3 -Wall
RUN gcc -o embedded-python embedded-python.o cJSON.o -L/usr/lib -lpython3.6m -ldl  -lm  -Xlinker -export-dynamic

FROM python:3.6.8-alpine3.8
COPY --from=0 /embed-work/embedded-python /bin/embedded-python
CMD ["/bin/embedded-python"]
