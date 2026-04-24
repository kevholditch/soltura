import fs from 'node:fs'
import path from 'node:path'

const repoRoot = path.resolve(process.cwd(), '..')
const fixturePath = process.env.TEST_FIXTURE_PATH || path.resolve(repoRoot, 'testdata/llm/default.json')
const fixture = JSON.parse(fs.readFileSync(fixturePath, 'utf8'))

function scriptFor(purpose) {
  const script = fixture?.purposes?.[purpose]
  if (!script) {
    throw new Error(`Missing purpose ${purpose} in fixture ${fixturePath}`)
  }
  return script
}

function stepFor(purpose, index = 0) {
  const script = scriptFor(purpose)
  if (Array.isArray(script.sequence) && script.sequence.length > 0) {
    if (!script.sequence[index]) {
      throw new Error(`Missing step ${index} for purpose ${purpose} in fixture ${fixturePath}`)
    }
    return script.sequence[index]
  }
  return script
}

export function completeText(purpose, index = 0) {
  const step = stepFor(purpose, index)
  if (typeof step.complete_text === 'string' && step.complete_text.length > 0) {
    return step.complete_text
  }
  if (Array.isArray(step.stream_chunks) && step.stream_chunks.length > 0) {
    return step.stream_chunks.join('')
  }
  throw new Error(`No complete_text or stream_chunks for purpose ${purpose} step ${index}`)
}

export function streamText(purpose, index = 0) {
  const step = stepFor(purpose, index)
  if (Array.isArray(step.stream_chunks) && step.stream_chunks.length > 0) {
    return step.stream_chunks.join('')
  }
  if (typeof step.complete_text === 'string' && step.complete_text.length > 0) {
    return step.complete_text
  }
  throw new Error(`No stream_chunks or complete_text for purpose ${purpose} step ${index}`)
}

export function correctionArray() {
  return JSON.parse(completeText('correction_analysis'))
}

