FROM node:alpine3.10

RUN apk add hugo git
RUN npm install -g postcss-cli autoprefixer postcss
EXPOSE 1313