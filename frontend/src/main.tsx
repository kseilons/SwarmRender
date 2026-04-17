import React from 'react'
import ReactDOM from 'react-dom/client'
import { AuthProvider } from "react-oidc-context";
import { userManager } from "./services/auth.ts";
import App from './App.tsx'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AuthProvider userManager={userManager}>
      <App />
    </AuthProvider>
  </React.StrictMode>,
)
