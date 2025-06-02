#syntax=docker/dockerfile:1

# Multi-stage Dockerfile for Node.js application
# First Stage: Use DHI Node.js dev image for build
FROM docker/dhi-node:18-dev AS build-stage

# Set environment variable for production
ENV NODE_ENV=production

# Set the working directory inside the build container
WORKDIR /usr/src/app

# Use cache for faster rebuilds of dependencies stage
COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --omit=dev

# Second Stage: Use DHI Node.js runtime image
FROM docker/dhi-node:18 AS runtime-stage

# Set the working directory inside the runtime container
WORKDIR /usr/src/app

# Copy only necessary files from build stage
COPY --from=build-stage /usr/src/app/node_modules ./node_modules
COPY . .

# Application running command
CMD ["index.js"]