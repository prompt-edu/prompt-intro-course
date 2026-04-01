import '@tumaet/prompt-ui-components/index.css'
import './screenshot.css'
import React from 'react'
import { createRoot } from 'react-dom/client'
import { ScreenshotApp } from './ScreenshotApp'

const root = document.getElementById('template-root')!
createRoot(root).render(
  <React.StrictMode>
    <ScreenshotApp />
  </React.StrictMode>,
)
