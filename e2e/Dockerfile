# Playwright runner image. The tag MUST match @playwright/test in package.json
# so the bundled browsers line up with the test runner.
FROM mcr.microsoft.com/playwright:v1.61.1-jammy

WORKDIR /work

# Install deps first for layer caching.
COPY package.json ./
RUN npm install --no-audit --no-fund

# Bake the test sources into the image. The compose file mounts only the
# report/results dirs back out to the host.
COPY . .

# Typecheck at build time: Playwright transpiles without checking, so a type error
# would otherwise only surface as a runtime failure deep into a suite run.
RUN npx tsc --noEmit

# `docker compose run --rm e2e-runner` reads this exit code.
CMD ["npx", "playwright", "test"]
