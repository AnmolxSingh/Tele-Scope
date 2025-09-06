# Dockerfile
FROM golang:1.23-alpine

WORKDIR /app

# Copy dependency files and download them
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of your source code
COPY . .

# Compile the application
RUN go build -o /order-service ./

# Expose the port your app runs on
EXPOSE 8080

# The command to run when the container starts
CMD [ "/order-service" ]