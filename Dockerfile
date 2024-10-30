FROM golang:latest
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go mod download
ENV POSTGRES_USER=
ENV POSTGRES_PASSWORD=
ENV POSTGRES_HOST=
ENV POSTGRES_PORT=
ENV POSTGRES_DB=
CMD ["go run app/main.go"]
