import { expect, test } from '@playwright/test'
import { completeText, correctionArray, streamText } from './helpers/fixture.js'

test('conversation flow streams assistant text and renders corrections', async ({ page }) => {
  const seedText = completeText('session_seed')
  const assistantReply = streamText('conversation_stream')
  const correction = correctionArray()[0]

  await page.goto('/')

  await page.getByPlaceholder(/My weekend plans/i).fill('Holidays')
  await page.getByRole('button', { name: 'Start Session' }).click()

  await expect(page.getByText(seedText)).toBeVisible()

  await page.getByPlaceholder(/Write in Spanish/i).fill('Este fin de semana voy al skatepark')
  await page.getByRole('button', { name: /Send/ }).click()

  await expect(page.getByText(assistantReply)).toBeVisible()

  await page.getByLabel('Grammar corrections').click()

  await expect(page.getByText(correction.original)).toBeVisible()
  await expect(page.getByText(correction.corrected)).toBeVisible()
  await expect(page.getByText(correction.explanation)).toBeVisible()
})

