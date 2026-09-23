FROM node:22-alpine AS build

WORKDIR /src/dashboard-react

COPY dashboard-react/package.json dashboard-react/package-lock.json ./
RUN npm ci
COPY dashboard-react/ ./
# DocsPage imports these Markdown files as Vite raw assets during the image build.
COPY docs/ /src/docs/
RUN npm run build

FROM nginx:1.27-alpine

COPY --from=build /src/dashboard-react/dist /usr/share/nginx/html
COPY deployments/nginx.conf /etc/nginx/conf.d/default.conf
