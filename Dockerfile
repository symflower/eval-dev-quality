FROM ubuntu:noble
RUN apt-get update && apt-get install -y ca-certificates wget unzip git make && update-ca-certificates

WORKDIR /home/ubuntu/eval-dev-quality
COPY ./ ./
RUN chown -R ubuntu:ubuntu ./

USER ubuntu
RUN mkdir -p ~/.eval-dev-quality

# Install Maven
RUN wget https://archive.apache.org/dist/maven/maven-3/3.9.1/binaries/apache-maven-3.9.1-bin.tar.gz && \
	tar -xf apache-maven-3.9.1-bin.tar.gz -C ~/.eval-dev-quality/ && \
	rm apache-maven-3.9.1-bin.tar.gz
ENV PATH="${PATH}:/home/ubuntu/.eval-dev-quality/apache-maven-3.9.1/bin"

# Install Gradle
RUN wget https://services.gradle.org/distributions/gradle-8.0.2-bin.zip && \
	unzip gradle-8.0.2-bin.zip -d ~/.eval-dev-quality/ && \
	rm gradle-8.0.2-bin.zip
ENV PATH="${PATH}:/home/ubuntu/.eval-dev-quality/gradle-8.0.2/bin"

# Install Java
RUN wget https://corretto.aws/downloads/latest/amazon-corretto-11-x64-linux-jdk.tar.gz && \
	tar -xf amazon-corretto-11-x64-linux-jdk.tar.gz -C ~/.eval-dev-quality/ && \
	rm amazon-corretto-11-x64-linux-jdk.tar.gz
ENV JAVA_HOME="/home/ubuntu/.eval-dev-quality/amazon-corretto-11.0.23.9.1-linux-x64"
ENV PATH="${PATH}:${JAVA_HOME}/bin"

# Install Go
RUN wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz && \
	tar -xf go1.21.5.linux-amd64.tar.gz -C ~/.eval-dev-quality/ && \
	rm go1.21.5.linux-amd64.tar.gz
ENV PATH="${PATH}:/home/ubuntu/.eval-dev-quality/go/bin"
ENV PATH="${PATH}:/home/ubuntu/go/bin"

# Setup the evaluation

RUN make install-all
ENV PATH="${PATH}:/home/ubuntu/.eval-dev-quality/bin"
