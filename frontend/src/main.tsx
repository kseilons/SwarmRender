import React from 'react'
import ReactDOM from 'react-dom/client'
import { AuthProvider } from "react-oidc-context";
import App from './App.tsx'
import './index.css'

const oidcConfig = {
  authority: import.meta.env.VITE_AUTH_AUTHORITY || "http://localhost:8083/realms/render-farm",
  client_id: import.meta.env.VITE_AUTH_CLIENT_ID || "render-farm-frontend",
  redirect_uri: window.location.origin,
  onSigninCallback: () => {
    window.history.replaceState({}, document.title, window.location.pathname);
  },
};

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AuthProvider {...oidcConfig}>
      <App />
    </AuthProvider>
  </React.StrictMode>,
)
