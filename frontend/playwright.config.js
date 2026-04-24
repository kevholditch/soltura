import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from '@playwright/test'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const repoRoot = path.resolve(__dirname, '..')

const fixturePath = path.resolve(repoRoot, process.env.TEST_FIXTURE_PATH || 'testdata/llm/default.json')
const runId = Date.now()
const dbPath = path.resolve('/tmp', `soltura-e2e-${runId}.sqlite`)
const port = process.env.PORT || '8090'
const baseURL = `http://127.0.0.1:${port}`

process.env.TEST_FIXTURE_PATH = fixturePath

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  webServer: {
    command: 'go run .',
    cwd: repoRoot,
    url: baseURL,
    timeout: 120_000,
    reuseExistingServer: false,
    env: {
      ...process.env,
      PORT: String(port),
      DB_PATH: dbPath,
      LLM_BACKEND: 'test',
      TEST_FIXTURE_PATH: fixturePath,
    },
  },
})

