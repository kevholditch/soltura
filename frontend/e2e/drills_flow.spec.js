import { expect, test } from '@playwright/test'
import { completeText, streamText } from './helpers/fixture.js'

function questionLead(text) {
  return text.split(/_{3,}/)[0].trim()
}

test('drills flow handles wrong then correct answer and reaches all done', async ({ page }) => {
  const drillStart = JSON.parse(completeText('drill_start'))
  const firstFeedback = streamText('drill_feedback_stream', 0)
  const secondFeedback = streamText('drill_feedback_stream', 1)
  const firstEval = JSON.parse(completeText('drill_evaluate', 0))

  await page.goto('/')

  // Prime vocab with one conversation turn so drill start has data.
  await page.getByPlaceholder(/My weekend plans/i).fill('Prime drills')
  await page.getByRole('button', { name: 'Start Session' }).click()
  await expect(page.getByText(completeText('session_seed'))).toBeVisible()
  await page.getByPlaceholder(/Write in Spanish/i).fill('Voy a ir a el skatepark')
  await page.getByRole('button', { name: /Send/ }).click()
  await expect(page.getByText(streamText('conversation_stream'))).toBeVisible()

  await page.getByLabel('Drills').click()

  await expect(page.getByText(drillStart.explanation)).toBeVisible()
  await expect(page.getByText(questionLead(drillStart.question), { exact: false })).toBeVisible()

  await page.getByLabel('Blank 1').fill('a el')
  await page.getByRole('button', { name: 'Submit' }).click()

  await expect(page.getByText(firstFeedback)).toBeVisible()
  await expect(page.getByText(questionLead(firstEval.next_question), { exact: false })).toBeVisible()

  await page.getByLabel('Blank 1').fill('al')
  await page.getByRole('button', { name: 'Submit' }).click()

  await expect(page.getByText(secondFeedback).first()).toBeVisible()
  await expect(page.getByText('✓ Dominado')).toBeVisible()

  await expect(page.getByText('All done!')).toBeVisible({ timeout: 10_000 })
})
