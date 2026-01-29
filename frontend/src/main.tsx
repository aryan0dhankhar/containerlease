import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'

console.log('main.tsx loaded')

const root = document.getElementById('root')
if (!root) {
  console.error('Root element not found!')
  document.body.innerHTML = '<div style="color: red; padding: 20px;">Root element not found</div>'
} else {
  console.log('Root element found, mounting React')
  try {
    ReactDOM.createRoot(root).render(
      <React.StrictMode>
        <App />
      </React.StrictMode>,
    )
    console.log('React mounted successfully')
  } catch (err) {
    console.error('React render failed:', err)
    root.innerHTML = `<div style="color: red; padding: 20px; font-family: monospace;">React Error: ${err instanceof Error ? err.message : String(err)}</div>`
  }
}
