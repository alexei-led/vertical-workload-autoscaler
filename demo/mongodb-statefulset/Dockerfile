# Dockerfile
FROM openjdk:8-jdk-alpine
RUN apk add --no-cache maven git python
WORKDIR /app
RUN git clone https://github.com/brianfrankcooper/YCSB.git
WORKDIR /app/YCSB
RUN mvn clean package -DskipTests
CMD [ "./bin/ycsb" ]