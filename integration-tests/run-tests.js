#!/usr/bin/env node
/**
 * Starts the backend with -configdir pointing to integration-tests/tests,
 * runs mocha, then stops the backend.
 * Run from repo root: node integration-tests/run-tests.js
 * Or: cd integration-tests && npm run test:run
 */
import { spawn } from 'child_process'
import { fileURLToPath } from 'url'
import { dirname, join } from 'path'
import { createInterface } from 'readline'

const __dirname = dirname(fileURLToPath(import.meta.url))
const repoRoot = join(__dirname, '..')
const testsDir = join(__dirname, 'tests')
const serviceBin = join(repoRoot, 'service', 'easypour-service')

async function main() {
  const env = { ...process.env, EASYPOUR_BASE_URL: 'http://localhost:9654' }
  const child = spawn(serviceBin, ['-configdir', testsDir], {
    cwd: join(repoRoot, 'service'),
    env,
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  let resolved = false
  const waitForServer = new Promise((resolve, reject) => {
    const rl = createInterface({ input: child.stdout })
    rl.on('line', (line) => {
      if (line.includes('Starting EasyPour') && !resolved) {
        resolved = true
        rl.close()
        resolve()
      }
    })
    child.stderr.on('data', (d) => {
      if (d.toString().includes('Starting EasyPour') && !resolved) {
        resolved = true
        resolve()
      }
    })
    child.on('error', (err) => !resolved && reject(err))
    setTimeout(() => {
      if (!resolved) {
        resolved = true
        resolve()
        // assume server is up after 2s
      }
    }, 2000)
  })
  await waitForServer
  const mocha = spawn('npx', ['mocha', '--timeout', '30000', 'tests/init.spec.js'], {
    cwd: __dirname,
    env: { ...process.env, EASYPOUR_BASE_URL: 'http://localhost:9654' },
    stdio: 'inherit',
  })
  const code = await new Promise((resolve) => mocha.on('close', resolve))
  child.kill('SIGTERM')
  process.exit(code)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
