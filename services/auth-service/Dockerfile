FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./
RUN npm ci --only=production

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Create secrets directory
RUN mkdir -p /app/secrets

# Set proper permissions
RUN chown -R node:node /app
USER node

EXPOSE 8080

CMD ["npm", "start"]
