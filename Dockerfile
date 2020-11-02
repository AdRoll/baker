FROM node:alpine3.10

RUN apk add hugo git

WORKDIR /baker
ADD . /baker
RUN rm ./themes/docsy -rf
RUN git submodule add --force https://github.com/google/docsy.git themes/docsy
RUN git submodule update --init --recursive
RUN npm install -g postcss-cli autoprefixer postcss

EXPOSE 1313
ENTRYPOINT ["hugo", "server", "--bind=0.0.0.0"]
