/**
 * E2E Playwright screenshots — captures all views from the real React app.
 * Requires: Go dev server on :8082, Rspack dev server on :3006
 */
const { chromium } = require('playwright')

const BASE = 'http://localhost:3006'
const OUT = '../.github/pr-screenshots'

const VIEWS = [
  // Seat grid — tutor view (default)
  { route: '/grid', name: 'screenshot-seat-grid-tutor', width: 1280, height: 900 },
  // Seat grid — peer group view (click button)
  { route: '/grid', name: 'screenshot-seat-grid-peer', width: 1280, height: 900, clickButton: 'Peer Group' },
  // Seat grid — seat view (click button)
  { route: '/grid', name: 'screenshot-seat-grid-seat', width: 1280, height: 900, clickButton: 'Seat' },
  // Admin tutor assignment table
  { route: '/tutor-table', name: 'screenshot-admin-table', width: 1100, height: 800 },
  // Admin peer review groups
  { route: '/peer-groups', name: 'screenshot-admin-peer-groups', width: 1100, height: 900 },
  // Empty state (no peer assignments)
  { route: '/empty-state', name: 'screenshot-empty-state', width: 900, height: 500 },
  // Student seat assignment view
  { route: '/student-view', name: 'screenshot-student-view', width: 1000, height: 700 },
]

async function main() {
  const browser = await chromium.launch()

  for (const view of VIEWS) {
    const context = await browser.newContext({
      viewport: { width: view.width, height: view.height },
      deviceScaleFactor: 2,
    })
    const page = await context.newPage()

    // Navigate using hash routing
    await page.goto(`${BASE}/#${view.route}`)
    // Wait for content to render (API fetch + React render)
    await page.waitForTimeout(2000)

    // Click a view mode button if needed
    if (view.clickButton) {
      const btn = page.locator('button', { hasText: view.clickButton })
      if (await btn.count() > 0) {
        await btn.click()
        await page.waitForTimeout(500)
      }
    }

    await page.screenshot({
      path: `${OUT}/${view.name}.png`,
      fullPage: true,
    })
    console.log(`✓ ${view.name}`)
    await context.close()
  }

  await browser.close()
  console.log(`\nAll ${VIEWS.length} screenshots saved to ${OUT}/`)
}

main().catch(e => { console.error(e); process.exit(1) })
