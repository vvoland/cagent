# syntax=docker/dockerfile:1

# Build stage
FROM docker/dhi-node:22.16-dev AS build-stage

# Set environment to production
ENV NODE_ENV=production

# Set the working directory inside the container
WORKDIR /usr/src/app

# Copy package.json and package-lock.json (if available)
COPY package*.json ./

# Install dependencies
RUN --mount=type=cache,target=/root/.npm npm ci --omit=dev

# Copy the application code
COPY . .

# Runtime stage
FROM docker/dhi-node:22.16 AS runtime-stage

# Set environment to production
ENV NODE_ENV=production

# Set the working directory inside the container
WORKDIR /usr/src/app

# Copy necessary files from the build stage
COPY --from=build-stage /usr/src/app/node_modules ./node_modules
COPY . .

# Expose the application port
EXPOSE 3000

# Command to run the Node.js application
CMD ["node", "index.js"]