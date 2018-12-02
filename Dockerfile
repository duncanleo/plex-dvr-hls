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
RUN apk add libva-intel-driver libva-utils --update-cache --repository http://dl-3.alpinelinux.org/alpine/edge/testing/ --allow-untrusted
COPY . .
COPY --from=deps /app/node_modules ./node_modules
RUN yarn
CMD [ "yarn", "start" ]
