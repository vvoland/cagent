# Migration Report: Docker Hardened Images (DHI)

## Summary of Changes

I've migrated the Dockerfile to use Docker Hardened Images (DHI) following best practices. The key changes include:

1. **Multi-Stage Build Pattern**:
   - Build stage using `docker/dhi-node:18-dev`
   - Runtime stage using `docker/dhi-node:18`

2. **Production Configuration**:
   - Set `NODE_ENV=production` for both stages
   - Used `npm ci --omit=dev` to optimize dependencies

3. **Security Improvements**:
   - Runtime container runs as a non-root user (provided by DHI)
   - Minimized image with only necessary files

## Dockerfile Comparison

### Original Dockerfile:
```dockerfile
#syntax=docker/dockerfile:1

# Use the official Node.js image from the Docker Hub
FROM node:latest

# Set the working directory inside the container
WORKDIR /usr/src/app

# Copy package.json and package-lock.json (if available)
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy the application code
COPY . .

# Command to run the Node.js application
CMD ["node", "index.js"]
```

### New Dockerfile (DHI):
```dockerfile
#syntax=docker/dockerfile:1

# Build stage with development image
FROM docker/dhi-node:18-dev AS build-stage

ENV NODE_ENV=production
WORKDIR /usr/src/app

# Copy package files and install dependencies
COPY package*.json ./
RUN npm ci --omit=dev

# Runtime stage with the hardened image
FROM docker/dhi-node:18 AS runtime-stage

ENV NODE_ENV=production
WORKDIR /usr/src/app

# Copy node_modules and application code from build stage
COPY --from=build-stage /usr/src/app/node_modules ./node_modules
COPY index.js ./

# Command to run the Node.js application
CMD ["index.js"]
```

## Key Improvements

1. **Security**: The DHI runtime image includes security patches and runs as a non-root user.

2. **Size Optimization**: The multi-stage build ensures only necessary files are included in the final image.

3. **Dependency Management**: Using `npm ci --omit=dev` ensures consistent, production-only dependencies.

4. **Best Practices**: Following DHI guidelines, removed the explicit `node` command from CMD.

## Usage Instructions

1. Replace your current Dockerfile with the new Dockerfile.dhi:
   ```bash
   mv Dockerfile.dhi Dockerfile
   ```

2. Build and run your container as usual:
   ```bash
   docker build -t your-app-name .
   docker run -p 3000:3000 your-app-name
   ```