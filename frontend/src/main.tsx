import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource/anton/400.css'
import '@fontsource/archivo/400.css'
import '@fontsource/space-mono/400.css'
import '@fontsource/space-mono/700.css'
import './index.css'
import App from './App.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
