FROM openjdk:13-alpine

COPY . /usr/src
WORKDIR /usr/src
RUN javac -Xlint:deprecation -cp .:json-20180813.jar ServerlessFunction.java StringClassLoader.java
CMD ["java", "-cp", ".:/usr/src/json-20180813.jar", "ServerlessFunction"]
