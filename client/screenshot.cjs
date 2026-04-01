/**
 * Take real Playwright screenshots of the SeatGrid component.
 * Run: node screenshot.cjs (with webpack.screenshot dev server on port 3006)
 */
const { chromium } = require('playwright')

const BASE = 'http://localhost:3006/course/c1/phase/p1'
const OUT = '../.github/pr-screenshots'

async function main() {
  const browser = await chromium.launch()
  const context = await browser.newContext({
    viewport: { width: 1280, height: 900 },
    deviceScaleFactor: 2,
  })

  // 1. Tutor view (default)
  const page1 = await context.newPage()
  await page1.goto(BASE)
  await page1.waitForSelector('button', { timeout: 10000 })
  await page1.waitForTimeout(500)
  await page1.screenshot({ path: `${OUT}/screenshot-seat-grid-tutor.png`, fullPage: true })
  console.log('✓ Tutor view screenshot')

  // 2. Peer Group view
  const page2 = await context.newPage()
  await page2.goto(BASE)
  await page2.waitForSelector('button', { timeout: 10000 })
  await page2.waitForTimeout(300)
  const peerBtn = page2.locator('button', { hasText: 'Peer Group' })
  await peerBtn.click()
  await page2.waitForTimeout(300)
  await page2.screenshot({ path: `${OUT}/screenshot-seat-grid-peer.png`, fullPage: true })
  console.log('✓ Peer Group view screenshot')

  // 3. Seat view
  const page3 = await context.newPage()
  await page3.goto(BASE)
  await page3.waitForSelector('button', { timeout: 10000 })
  await page3.waitForTimeout(300)
  const seatBtn = page3.locator('button', { hasText: 'Seat' })
  await seatBtn.click()
  await page3.waitForTimeout(300)
  await page3.screenshot({ path: `${OUT}/screenshot-seat-grid-seat.png`, fullPage: true })
  console.log('✓ Seat view screenshot')

  await browser.close()
  console.log(`\nAll screenshots saved to ${OUT}/`)
}

main().catch(e => { console.error(e); process.exit(1) })
