
import { expect, test } from '@playwright/test'
import { correctionArray, completeText } from './helpers/fixture.js'

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
  await expect(page.getByText(correction.original)).toBeVisible()
  await expect(page.getByText(correction.corrected)).toBeVisible()
  await expect(page.getByText(correction.category)).toBeVisible()
})
