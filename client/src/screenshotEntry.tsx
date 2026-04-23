import './screenshot.css'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { ScreenshotApp } from './ScreenshotApp'

// Pass the URL hash as the initial route so Playwright can navigate
// e.g. http://localhost:3006/#/peer-groups → initialRoute="/peer-groups"
const initialRoute = window.location.hash.replace('#', '') || '/grid'

const root = document.getElementById('template-root')!
createRoot(root).render(
  <React.StrictMode>
    <ScreenshotApp initialRoute={initialRoute} />
  </React.StrictMode>,
)
