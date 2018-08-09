FROM node:10

WORKDIR /gqlgen/integration


COPY integration/package*.json /gqlgen/integration/
RUN npm ci

COPY . /gqlgen/
