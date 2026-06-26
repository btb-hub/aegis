import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { I18nextProvider } from 'react-i18next';
import { App } from './App';
import './index.css';
import i18n, { resolveLocale } from './i18n';

void i18n.changeLanguage(resolveLocale());
document.documentElement.lang = i18n.language.startsWith('ru') ? 'ru' : 'en';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(
    <StrictMode>
      <I18nextProvider i18n={i18n}>
        <App />
      </I18nextProvider>
    </StrictMode>,
  );
}
