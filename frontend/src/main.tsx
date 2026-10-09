import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App'
import './theme.css'
import './styles.css'
import './lists.css'
import './dialogs.css'
import './visitron.css'

const root = document.getElementById('root')
if (root) {
  ReactDOM.createRoot(root).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>,
  )
}
