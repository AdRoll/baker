FROM klakegg/hugo:ext-alpine

RUN npm install -g postcss-cli autoprefixer postcss

EXPOSE 1313
