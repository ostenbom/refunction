FROM openjdk:13-alpine

COPY . /usr/src
WORKDIR /usr/src
RUN javac -Xlint:deprecation -cp .:gson.jar ServerlessFunction.java StringClassLoader.java
CMD ["java", "-cp", ".:/usr/src/gson.jar", "ServerlessFunction"]
