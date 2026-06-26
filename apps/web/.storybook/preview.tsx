import type { Preview } from '@storybook/react';
import { I18nextProvider } from 'react-i18next';
import '../src/index.css';
import i18n from '../src/i18n';

const preview: Preview = {
  parameters: {
    layout: 'padded',
    controls: { matchers: { color: /(background|color)$/i, date: /Date$/i } },
  },
  globalTypes: {
    locale: {
      name: 'Locale',
      description: 'App locale',
      defaultValue: 'en',
      toolbar: {
        icon: 'globe',
        items: [
          { value: 'en', title: 'English' },
          { value: 'ru', title: 'Русский' },
        ],
      },
    },
  },
  decorators: [
    (Story, context) => {
      const locale = context.globals.locale === 'ru' ? 'ru' : 'en';
      void i18n.changeLanguage(locale);
      document.documentElement.lang = locale;
      return (
        <I18nextProvider i18n={i18n}>
          <Story />
        </I18nextProvider>
      );
    },
  ],
};

export default preview;
