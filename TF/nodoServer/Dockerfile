# Dockerfile for Server

FROM golang:alpine

# Set the working directory
WORKDIR /app

# Copy the source code to the container
COPY ./codeServer.go .

# Expose the ports
EXPOSE 9002
EXPOSE 8080

# Run the server
CMD ["go", "run", "codeServer.go"]
