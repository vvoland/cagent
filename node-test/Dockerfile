#syntax=docker/dockerfile:1

# Use the official Node.js image from the Docker Hub
FROM node:latest

# Set the working directory inside the container
WORKDIR /usr/src/app

COPY package*.json ./

# Install dependencies
RUN npm installAdd commentMore actions

# Copy the application code
COPY . .

# Command to run the Node.js application
CMD ["node", "index.js"]
