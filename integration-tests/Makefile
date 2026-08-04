.PHONY: test install

# Run integration tests (backend must be built and will be started with -configdir)
test: install
	npm run test

# Run tests with backend lifecycle (start backend, run mocha, stop backend)
test:run: install
	node run-tests.js

install:
	npm install
