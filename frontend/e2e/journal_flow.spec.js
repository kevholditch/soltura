import { expect, test } from '@playwright/test'
import { completeText, correctionArray, streamText } from './helpers/fixture.js'

test('journal opens completed session with read-only transcript review corrections and summary', async ({ page }) => {
  const topic = 'Journal evidence session'
  const userTurn = 'Voy a ir a el skatepark'
  const seedText = completeText('session_seed')
  const assistantReply = streamText('conversation_stream')
  const correction = correctionArray()[0]
  const expectedFocusText = completeText('session_summary')
    .replaceAll('\\n', '\n')
    .match(/Focus on contractions[^\n]*/)?.[0]
    ?.replaceAll('**', '')
    .trim()
  expect(expectedFocusText).toBeTruthy()

  await page.goto('/')

  await page.getByPlaceholder(/My weekend plans/i).fill(topic)
  await page.getByRole('button', { name: 'Start Session' }).click()
  await expect(page.getByText(seedText)).toBeVisible()

  await page.getByPlaceholder(/Write in Spanish/i).fill(userTurn)
  await page.getByRole('button', { name: /Send/ }).click()
  await expect(page.getByText(assistantReply)).toBeVisible()

  await page.getByRole('button', { name: 'End Session' }).click()
  await expect(page.getByRole('heading', { name: 'Session Complete' })).toBeVisible()

  await page.getByLabel('Journal').click()

  await expect(page.getByRole('heading', { name: 'Journal' })).toBeVisible()
  await expect(page.getByRole('heading', { name: /^Today$/i })).toBeVisible()
  await expect(page.getByText(/\d+\s+sessions?/i).first()).toBeVisible()

  const sessionEntry = page
    .getByRole('button', { name: new RegExp(topic, 'i') })
    .or(page.getByRole('link', { name: new RegExp(topic, 'i') }))
  await sessionEntry.click()

  await expect(page.getByRole('heading', { name: topic })).toBeVisible()
  await expect(page.getByText(seedText)).toBeVisible()
  await expect(page.getByText(userTurn, { exact: true })).toBeVisible()
  await expect(page.getByText(assistantReply)).toBeVisible()

  await expect(page.getByRole('heading', { name: 'Review', exact: true })).toBeVisible()
  await expect(page.getByText('What went well')).toBeVisible()
  await expect(page.getByText('Focus on contractions', { exact: false })).toBeVisible()
  await expect(page.getByText(expectedFocusText, { exact: false })).toBeVisible()

  await expect(page.getByRole('heading', { name: 'Corrections', exact: true })).toBeVisible()
  const correctionsSection = page.getByLabel('Corrections')
  await expect(correctionsSection.getByText(correction.original)).toBeVisible()
  await expect(correctionsSection.getByText(correction.corrected)).toBeVisible()
  await expect(correctionsSection.getByText(correction.explanation)).toBeVisible()

  await expect(page.getByPlaceholder(/Write in Spanish/i)).toHaveCount(0)
  await expect(page.getByRole('button', { name: /Send/ })).toHaveCount(0)
})

test('journal ignores sessions ended before the learner responds', async ({ page }) => {
  const topic = 'Seed only journal session'

  await page.goto('/')

  await page.getByPlaceholder(/My weekend plans/i).fill(topic)
  await page.getByRole('button', { name: 'Start Session' }).click()
  await expect(page.getByText(completeText('session_seed'))).toBeVisible()

  await page.getByRole('button', { name: 'End Session' }).click()
  await expect(page.getByRole('heading', { name: 'Session Complete' })).toBeVisible()

  await page.getByLabel('Journal').click()

  await expect(page.getByRole('heading', { name: 'Journal' })).toBeVisible()
  await expect(page.getByText(topic)).toHaveCount(0)
})
