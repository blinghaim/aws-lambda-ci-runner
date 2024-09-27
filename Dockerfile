FROM golang:1.22.2 as build
WORKDIR /helloworld
COPY go.mod ./
COPY Main.go ./main.go
RUN go mod tidy
RUN go build -tags lambda.norpc -o main main.go
FROM amazonlinux:2023.5.20240903.0
RUN yum -y groupinstall "Development Tools"
COPY --from=build /usr/local/go/ /usr/local/go/
RUN curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
ENV NVM_DIR="/root/.nvm"
RUN [[ -s "/root/.nvm/nvm.sh" ]] && \. "/root/.nvm/nvm.sh" && nvm install 18.20.4 && npm install --global yarn typescript next react react-dom -f
RUN cp -rf /root/.nvm/versions/node/v18.20.4 /usr/local/node
COPY --from=build /helloworld/main ./main
RUN mkdir -p /root/.cache
RUN chmod 777 /root/.cache
RUN mkdir -p /mnt/.cache
RUN chmod -R 777 /mnt
ENV PATH /usr/local/node/bin:/usr/local/bin:/usr/bin/:/bin:/usr/local/go/bin:/usr/local/sbin:/usr/sbin:/sbin
USER 0:0
ENTRYPOINT [ "./main" ]
