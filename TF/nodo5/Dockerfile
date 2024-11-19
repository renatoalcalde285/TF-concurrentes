# Dockerfile for Node

FROM golang:alpine

# Set the working directory
WORKDIR /app

# Copy the source code to the container
COPY ./codeRedP2PHP.go .

# Expose the port
EXPOSE 9002

# Run the node
CMD ["go", "run", "codeRedP2PHP.go"]
