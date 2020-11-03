FROM baker-docs:base

ADD . /baker
WORKDIR /baker

EXPOSE 1313
ENTRYPOINT ["hugo", "server", "--minify", "--bind=0.0.0.0"]