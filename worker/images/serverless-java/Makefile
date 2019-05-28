.PHONY: default
default:
	docker build -t ostenbom/serverless-java .
	docker create --name dummy ostenbom/serverless-java sh
	docker cp dummy:/usr/src/ServerlessFunction.class ServerlessFunction.class
	docker cp dummy:/usr/src/StringClassLoader.class StringClassLoader.class
	docker container rm dummy

local:
	javac -cp .:gson.jar ServerlessFunction.java StringClassLoader.java

clean:
	rm -rf *.class
