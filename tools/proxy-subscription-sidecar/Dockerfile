FROM node:22-alpine
WORKDIR /app
COPY package.json ./
COPY server.mjs ./
RUN apk add --no-cache sing-box
EXPOSE 8080
CMD ["node", "server.mjs"]
