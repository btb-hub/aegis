export const settingsFixture = {
  values: { PUBLIC_URL: 'https://aegis.example', HTTP_ADDR: ':8080', WEBHOOK_SECRET: '', ADMIN_EMAILS: 'admin@example.com', SESSION_TTL: '168h', INCIDENT_DEDUP_WINDOW: '24h', ESCALATION_TIMEOUT: '15m', ALERT_FINGERPRINT_LABELS: 'alertname,team' },
  revision: 1, enabled: { google: true }, secret_present: { WEBHOOK_SECRET: true }, bootstrap_closed: true,
};

// Runs before mounting a story and restores fetch when leaving it. No live service calls.
export function settingsStoryFixtures() {
  const original = window.fetch;
  window.fetch = async (input) => {
    const path = String(input);
    let body: unknown = settingsFixture;
    if (path === '/auth/me') {
      body = { id: 'storybook-admin', email: 'admin@example.com', display_name: 'Administrator', role: 'admin', locale: 'en', provider: 'google' };
    } else if (path.includes('/providers/')) {
      const provider = path.split('/providers/')[1].split('/')[0];
      const prefix = `${provider.toUpperCase()}_OIDC_`;
      body = { provider, values: { [`${prefix}CLIENT_ID`]: 'example-client', [`${prefix}CLIENT_SECRET`]: '', [`${prefix}REDIRECT_URL`]: `https://aegis.example/auth/${provider}/callback`, ...(provider === 'express' ? { [`${prefix}ISSUER`]: 'https://sso.example' } : {}) }, secret_present: { [`${prefix}CLIENT_SECRET`]: true }, revision: 1, tested: false, expected_email: 'admin@example.com' };
    } else if (path !== '/api/v1/settings') {
      body = { items: [] };
    }
    return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } });
  };
  return () => { window.fetch = original; };
}
