// Minimal stand-in for Google Identity Services, served in place of the real
// GSI script during e2e. Implements only the OAuth2 code-client slice the
// login page uses: requestCode() fires the callback with a sentinel code.
window.google = {
  accounts: {
    oauth2: {
      initCodeClient(cfg) {
        return {
          requestCode() {
            const email = window.__E2E_EMAIL__ || 'e2e@gmail.com';
            cfg.callback({ code: 'e2e:' + email });
          },
        };
      },
    },
  },
};
