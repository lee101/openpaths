import {StrictMode} from 'react';
import {createRoot} from 'react-dom/client';
import App from './App.tsx';
import './index.css';
import {installFrontendErrorReporting} from './lib/frontendErrors.ts';

installFrontendErrorReporting();

// Older sessions may contain an API key, while existing accounts receive a JWT.
// Never replace a persisted API key with a dashboard-only JWT.
if (
  typeof window !== 'undefined' &&
  !localStorage.getItem('op_api_key') &&
  (window.userData?.secret?.startsWith('sk-op-') || window.userData?.secret?.startsWith('op_'))
) {
  localStorage.setItem('op_api_key', window.userData.secret);
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
