.PHONY: default
default:
	docker build -t ostenbom/embedded-python .
	docker create --name dummy ostenbom/embedded-python sh
	docker cp dummy:/bin/embedded-python embedded-python
	docker container rm dummy

local:
	gcc -c embedded-python.c cJSON.c -I/usr/include/python3.6m -I/usr/include/python3.6m  -Wno-unused-result -Wsign-compare -g -fdebug-prefix-map=/build/python3.6-A7ntPm/python3.6-3.6.7=. -specs=/usr/share/dpkg/no-pie-compile.specs -fstack-protector -Wformat -Werror=format-security  -DNDEBUG -g -fwrapv -O3 -Wall -fPIC
	gcc -o local-embedded-python embedded-python.o cJSON.o -L/usr/lib/python3.6/config-3.6m-x86_64-linux-gnu -L/usr/lib -lpython3.6m -lpthread -ldl  -lutil -lm  -Xlinker -export-dynamic -Wl,-O1 -Wl,-Bsymbolic-functions

clean:
	rm -rf *.o embedded-python local-embedded-python
