package build

import "fmt"

// DockerfileTemplate returns the base Dockerfile content for a given runtime.
// These are written to a temp file inside the cloned repo before the image build.
func DockerfileTemplate(runtime Runtime) (string, error) {
	switch runtime {
	case RuntimeNode:
		return nodeDockerfile, nil
	case RuntimePython:
		return pythonDockerfile, nil
	case RuntimeGo:
		return goDockerfile, nil
	case RuntimeStatic:
		return staticDockerfile, nil
	default:
		return "", fmt.Errorf("no Dockerfile template for runtime %q", runtime)
	}
}

const nodeDockerfile = `FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
EXPOSE 3000
CMD ["npm", "start"]
`

const pythonDockerfile = `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["python", "app.py"]
`

const goDockerfile = `FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
`

const staticDockerfile = `FROM nginx:alpine
COPY . /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
`