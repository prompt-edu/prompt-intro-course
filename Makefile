.PHONY: test-e2e test-e2e-ui test-e2e-down

E2E_COMPOSE = docker compose -f docker-compose.e2e.yml --env-file e2e/.env.e2e

# Build the stack, run the containerized Playwright runner, then tear everything
# down — preserving the runner's exit code so CI and `make` fail when tests fail.
test-e2e:
	@mkdir -p e2e/playwright-report e2e/test-results
	@$(E2E_COMPOSE) build
	@$(E2E_COMPOSE) run --rm e2e-runner; status=$$?; \
		$(E2E_COMPOSE) down -v; \
		exit $$status

# Playwright's UI mode, served out of the runner container. Open it at
# http://127.0.0.1:8123 — not localhost, which resolves to ::1 first while the
# published port is IPv4. tests/ and src/ are bind-mounted so edits are picked up
# without rebuilding the image.
test-e2e-ui:
	@mkdir -p e2e/playwright-report e2e/test-results
	@$(E2E_COMPOSE) run --rm --build -p 8123:8123 \
		-v "$(CURDIR)/e2e/tests:/work/tests" \
		-v "$(CURDIR)/e2e/src:/work/src" \
		-v "$(CURDIR)/e2e/playwright-report:/work/playwright-report" \
		-v "$(CURDIR)/e2e/test-results:/work/test-results" \
		e2e-runner npx playwright test --ui-host=0.0.0.0 --ui-port=8123; \
		$(E2E_COMPOSE) down -v

test-e2e-down:
	@$(E2E_COMPOSE) down -v
