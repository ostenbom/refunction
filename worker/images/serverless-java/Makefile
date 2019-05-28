.PHONY: default
default:
	docker build -t ostenbom/serverless-java .
	docker create --name dummy ostenbom/serverless-java sh
	docker cp dummy:/usr/src/ServerlessFunction.class ServerlessFunction.class
	docker cp dummy:/usr/src/StringJarLoader.class StringJarLoader.class
	docker container rm dummy

local:
	javac -cp .:gson.jar ServerlessFunction.java StringJarLoader.java

clean:
	rm -rf *.class
