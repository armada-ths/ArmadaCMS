# Dockerfile References: https://docs.docker.com/engine/reference/builder/
# FROM node:23-slim AS frontend-builder
# RUN rm -rf ./frontend/.cache
# RUN rm -rf ./frontend/dist
# RUN rm -rf ./frontend/node_modules
# COPY frontend/package.json /admin/package.json
# COPY frontend/yarn.lock /admin/yarn.lock
# WORKDIR /admin
# RUN yarn install
# ADD frontend /admin
# RUN WPATH='/admin' yarn run build
# ----------------------
# Stage 1: Build React frontend
# ----------------------
FROM node:20-slim AS frontend-builder

WORKDIR /frontend

# Copy package files and install dependencies
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install --legacy-peer-deps

# Copy the rest of the frontend source and build
COPY frontend/ .
RUN npm run build


# Start from golang:1.12-alpine base image
FROM golang:1.24-alpine

# The latest alpine images don't have some tools like (`git` and `bash`).
# Adding git, bash and openssh to the image
RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

RUN go install github.com/air-verse/air@v1.61.1

# Set the Current Working Directory inside the container
WORKDIR /app
ENV PATH="/go/bin:${PATH}"

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependancies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .
#COPY --from=frontend-builder /admin/dist ./frontend/dist


COPY --from=frontend-builder /frontend/dist ./frontend/dist


# Expose port 8080 to the outside world
EXPOSE 8080

# Run the executable
ENTRYPOINT ["air"]