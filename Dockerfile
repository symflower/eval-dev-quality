# Builder image.
FROM golang:latest as builder

WORKDIR /app
COPY ./ ./

# Build the binary.
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o eval-dev-quality ./cmd/eval-dev-quality

# Actual running image.
FROM ubuntu:noble
RUN apt-get update && apt-get install -y ca-certificates wget unzip git make && update-ca-certificates

# Non-root ollama need a hardcoded directory to store ssh-key.
# RUN mkdir -p /.ollama && chmod 777 /.ollama
# # Same for symflower
# RUN mkdir -p /.config && chmod 777 /.config
# RUN mkdir -p /.eval-dev-quality && chmod 777 /.eval-dev-quality
# RUN mkdir -p /.cache && chmod 777 /.cache
# # Non-root go folder
# RUN mkdir -p /go && chmod 777 /go

# Switch to the ubuntu user as we want it to run as non-root.
#USER ubuntu
WORKDIR /app
COPY ./testdata ./testdata
COPY ./Makefile ./Makefile
RUN mkdir -p .eval-dev-quality

# Install Maven
RUN wget https://archive.apache.org/dist/maven/maven-3/3.9.1/binaries/apache-maven-3.9.1-bin.tar.gz && \
	tar -xf apache-maven-3.9.1-bin.tar.gz -C /app/.eval-dev-quality/ && \
	rm apache-maven-3.9.1-bin.tar.gz
ENV PATH="${PATH}:/app/.eval-dev-quality/apache-maven-3.9.1/bin"

# Install Gradle
RUN wget https://services.gradle.org/distributions/gradle-8.0.2-bin.zip && \
	unzip gradle-8.0.2-bin.zip -d /app/.eval-dev-quality/ && \
	rm gradle-8.0.2-bin.zip
ENV PATH="${PATH}:/app/.eval-dev-quality/gradle-8.0.2/bin"

# Install Java
RUN wget https://corretto.aws/downloads/latest/amazon-corretto-11-x64-linux-jdk.tar.gz && \
	tar -xf amazon-corretto-11-x64-linux-jdk.tar.gz -C /app/.eval-dev-quality/ && \
	rm amazon-corretto-11-x64-linux-jdk.tar.gz
ENV JAVA_HOME="/app/.eval-dev-quality/amazon-corretto-11.0.23.9.1-linux-x64"
ENV PATH="${PATH}:${JAVA_HOME}/bin"

# Install Go
RUN wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz && \
	tar -xf go1.21.5.linux-amd64.tar.gz -C /app/.eval-dev-quality/ && \
	rm go1.21.5.linux-amd64.tar.gz
ENV PATH="${PATH}:/app/.eval-dev-quality/go/bin"
ENV GOROOT="/app/.eval-dev-quality/go"
ENV GOPATH="/app/.eval-dev-quality/go"

# Install the binary
COPY --from=builder /app/eval-dev-quality /app/.eval-dev-quality/bin/
ENV PATH="${PATH}:/app/.eval-dev-quality/bin"
RUN make install-tools-testing
RUN make install-tools /app/.eval-dev-quality/bin

# CHMOD everything because of non-root user ids
RUN chmod -R 777 /app
