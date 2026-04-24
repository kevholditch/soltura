import { expect, test } from '@playwright/test'
import { completeText } from './helpers/fixture.js'

test('summary flow shows fixture-based summary after ending session', async ({ page }) => {
  const summaryText = completeText('session_summary')
  const expectedFocusText = summaryText
    .replaceAll('\\n', '\n')
    .match(/Focus on contractions[^\n]*/)?.[0]
    ?.replaceAll('**', '')
    .trim()
  expect(expectedFocusText).toBeTruthy()

  await page.goto('/')

  await page.getByPlaceholder(/My weekend plans/i).fill('Weekend plans')
  await page.getByRole('button', { name: 'Start Session' }).click()
  await expect(page.getByText(completeText('session_seed'))).toBeVisible()

  await page.getByPlaceholder(/Write in Spanish/i).fill('Voy a ir a el skatepark con mi hija')
  await page.getByRole('button', { name: /Send/ }).click()
  await expect(page.getByText('Suena genial.')).toBeVisible()

  await page.getByRole('button', { name: 'End Session' }).click()

  await expect(page.getByRole('heading', { name: 'Session Complete' })).toBeVisible()
  await expect(page.getByText(expectedFocusText, { exact: false })).toBeVisible()
  await expect(page.getByText('Turns').first()).toBeVisible()
  await expect(page.getByText('Corrections').first()).toBeVisible()
})
