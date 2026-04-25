
import { expect, test } from '@playwright/test'
import { correctionArray, completeText } from './helpers/fixture.js'

function questionLead(text) {
  return text.split(/_{3,}/)[0].trim()
}

test('vocabulary flow lists correction entries from prior conversation turns', async ({ page }) => {
  const correction = correctionArray()[0]

  await page.goto('/')

  await page.getByPlaceholder(/My weekend plans/i).fill('Spanish chat')
  await page.getByRole('button', { name: 'Start Session' }).click()
  await expect(page.getByText(completeText('session_seed'))).toBeVisible()

  await page.getByPlaceholder(/Write in Spanish/i).fill('Voy a ir a el skatepark')
  await page.getByRole('button', { name: /Send/ }).click()
  await expect(page.getByText(completeText('conversation_stream'))).toBeVisible()

  await page.getByLabel('Vocabulary').click()

  await expect(page.getByRole('heading', { name: 'Vocabulary' })).toBeVisible()
  await expect(page.getByRole('cell', { name: correction.original })).toBeVisible()
  await expect(page.getByRole('cell', { name: correction.corrected })).toBeVisible()
  await expect(page.getByText(correction.category).first()).toBeVisible()
})

test('vocabulary shows last seen focus topic evidence and starts a scoped drill', async ({ page }) => {
  const correction = correctionArray()[0]
  const drillStart = JSON.parse(completeText('drill_start'))
  const focusTitle = `${correction.original} -> ${correction.corrected}`

  await page.goto('/')

  await page.getByPlaceholder(/My weekend plans/i).fill('Vocabulary focus topic')
  await page.getByRole('button', { name: 'Start Session' }).click()
  await expect(page.getByText(completeText('session_seed'))).toBeVisible()

  await page.getByPlaceholder(/Write in Spanish/i).fill('Voy a ir a el skatepark')
  await page.getByRole('button', { name: /Send/ }).click()
  await expect(page.getByText(completeText('conversation_stream'))).toBeVisible()

  await page.getByLabel('Vocabulary').click()

  await expect(page.getByRole('heading', { name: 'Vocabulary' })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'Last seen' })).toBeVisible()
  await expect(page.getByRole('button', { name: focusTitle })).toBeVisible()

  await page.getByRole('button', { name: focusTitle }).click()

  await expect(page.getByRole('heading', { name: /evidence/i })).toBeVisible()
  await expect(page.getByText(correction.explanation)).toBeVisible()
  await expect(page.getByRole('button', { name: 'Start drill' })).toBeVisible()

  await page.getByRole('button', { name: 'Start drill' }).click()

  await expect(page.getByText(drillStart.explanation)).toBeVisible()
  await expect(page.getByText(questionLead(drillStart.question), { exact: false })).toBeVisible()
})
