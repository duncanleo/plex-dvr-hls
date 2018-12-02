FROM node:10-alpine AS deps

WORKDIR /app
COPY package.json .
COPY yarn.lock .

RUN apk --update add --virtual build_deps \
    build-base libc-dev linux-headers python
RUN yarn

FROM node:10-alpine
WORKDIR /app
RUN apk --update add ffmpeg
COPY . .
COPY --from=deps /app/node_modules ./node_modules
RUN yarn
CMD [ "yarn", "start" ]
