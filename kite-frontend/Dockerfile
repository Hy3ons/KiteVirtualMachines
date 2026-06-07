# Stage 1: Build the static files
FROM node:20-alpine as build
WORKDIR /app

ARG VITE_BUILD_MODE=production
ARG VITE_API_BASE_URL=/api/v1
ARG VITE_USE_MOCK=false
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_USE_MOCK=${VITE_USE_MOCK}

# Copy package.json and install dependencies
COPY package*.json ./
RUN npm install

# Copy the rest of the source code and build
COPY . .
RUN npm run build -- --mode "${VITE_BUILD_MODE}"

# Stage 2: Serve the static files with Nginx
FROM nginx:alpine

# Remove default nginx static assets
RUN rm -rf /usr/share/nginx/html/*

# Configure SPA route fallback so direct requests such as /dashboard serve index.html.
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy the static files from the build stage
COPY --from=build /app/dist /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]
