# syntax = docker/dockerfile:1

ARG BUN_VERSION=1.3.2
FROM oven/bun:${BUN_VERSION}-slim AS base

LABEL fly_launch_runtime="Bun"

WORKDIR /app

ENV NODE_ENV="production"


# Throw-away build stage to reduce size of final image
FROM base AS build

# Install packages needed to build node modules
RUN apt-get update -qq && \
    apt-get install --no-install-recommends -y build-essential pkg-config python-is-python3

# Copy everything first — fumadocs-mdx postinstall needs source.config.ts
COPY . .

# Install dependencies (postinstall runs fumadocs-mdx which reads source.config.ts)
RUN bun install

# Build application
RUN bun --bun run build

# Remove dev dependencies for production
RUN rm -rf node_modules && \
    bun install --ci


# Final stage for app image
FROM base

COPY --from=build /app /app

EXPOSE 3000
CMD [ "bun", "run", "start" ]
