// Basil auth components runtime.
// Binds the elements rendered by the Register, Login, and Logout components
// from @basil/auth. Loaded once per page (each component emits the script tag;
// the guard below makes extra copies harmless). Configuration travels on
// data-* attributes so this file can be static and cached.
(function () {
  if (window.__basilAuthInit) return;
  window.__basilAuthInit = true;

  // Base64url helpers (WebAuthn payloads)
  function base64ToBuffer(base64) {
    const str = atob(base64.replace(/-/g, '+').replace(/_/g, '/'));
    const bytes = new Uint8Array(str.length);
    for (let i = 0; i < str.length; i++) bytes[i] = str.charCodeAt(i);
    return bytes.buffer;
  }

  function bufferToBase64(buffer) {
    const bytes = new Uint8Array(buffer);
    let str = '';
    for (let i = 0; i < bytes.length; i++) str += String.fromCharCode(bytes[i]);
    return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
  }

  function showError(container, message) {
    const errorDiv = container.querySelector('.basil-auth-error');
    if (errorDiv) {
      errorDiv.textContent = message;
      errorDiv.hidden = false;
    }
  }

  function hideError(container) {
    const errorDiv = container.querySelector('.basil-auth-error');
    if (errorDiv) errorDiv.hidden = true;
  }

  function bindRegister(form) {
    const redirect = form.dataset.redirect || '/';
    const recoveryPage = form.dataset.recoveryPage || '';

    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      hideError(form);

      const name = form.querySelector('input[name="name"]').value;
      const emailInput = form.querySelector('input[name="email"]');
      const email = (emailInput && emailInput.value) || null;

      try {
        // Step 1: Begin registration
        const beginRes = await fetch('/__auth/register/begin', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name, email })
        });

        if (!beginRes.ok) {
          const err = await beginRes.json();
          throw new Error(err.error || 'Registration failed');
        }

        const beginData = await beginRes.json();
        const pubKeyOptions = beginData.options.publicKey;
        const challengeId = beginData.challenge_id;

        // Step 2: Browser creates credential
        const credential = await navigator.credentials.create({
          publicKey: {
            challenge: base64ToBuffer(pubKeyOptions.challenge),
            rp: pubKeyOptions.rp,
            user: {
              id: base64ToBuffer(pubKeyOptions.user.id),
              name: pubKeyOptions.user.name,
              displayName: pubKeyOptions.user.displayName
            },
            pubKeyCredParams: pubKeyOptions.pubKeyCredParams,
            authenticatorSelection: pubKeyOptions.authenticatorSelection,
            timeout: pubKeyOptions.timeout,
            attestation: pubKeyOptions.attestation
          }
        });

        // Step 3: Finish registration
        const finishRes = await fetch('/__auth/register/finish', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            challenge_id: challengeId,
            response: {
              id: credential.id,
              rawId: bufferToBase64(credential.rawId),
              type: credential.type,
              response: {
                clientDataJSON: bufferToBase64(credential.response.clientDataJSON),
                attestationObject: bufferToBase64(credential.response.attestationObject)
              }
            }
          })
        });

        if (!finishRes.ok) {
          const err = await finishRes.json();
          throw new Error(err.error || 'Registration failed');
        }

        const result = await finishRes.json();

        // Handle recovery codes
        if (result.recovery_codes) {
          if (recoveryPage) {
            sessionStorage.setItem('basil_recovery_codes', JSON.stringify(result.recovery_codes));
            sessionStorage.setItem('basil_recovery_user', (result.user && result.user.name) || name);
            window.location.href = recoveryPage;
            return;
          }
          alert('Save these recovery codes:\n\n' + result.recovery_codes.join('\n'));
        }

        // Redirect on success (if not already redirected by recovery page)
        window.location.href = redirect;
      } catch (err) {
        console.error('Registration error:', err);
        showError(form, err.message || 'Registration failed. Please try again.');
      }
    });
  }

  function bindLogin(container) {
    const button = container.querySelector('button');
    const redirect = container.dataset.redirect || '/';
    if (!button) return;

    button.addEventListener('click', async () => {
      hideError(container);

      try {
        // Step 1: Begin login
        const beginRes = await fetch('/__auth/login/begin', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' }
        });

        if (!beginRes.ok) {
          const err = await beginRes.json();
          throw new Error(err.error || 'Login failed');
        }

        const beginData = await beginRes.json();
        const pubKeyOptions = beginData.options.publicKey;
        const challengeId = beginData.challenge_id;

        // Step 2: Browser authenticates
        const credential = await navigator.credentials.get({
          publicKey: {
            challenge: base64ToBuffer(pubKeyOptions.challenge),
            rpId: pubKeyOptions.rpId,
            timeout: pubKeyOptions.timeout,
            userVerification: pubKeyOptions.userVerification,
            allowCredentials: (pubKeyOptions.allowCredentials || []).map((c) => ({
              id: base64ToBuffer(c.id),
              type: c.type
            }))
          }
        });

        // Step 3: Finish login
        const finishRes = await fetch('/__auth/login/finish', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            challenge_id: challengeId,
            response: {
              id: credential.id,
              rawId: bufferToBase64(credential.rawId),
              type: credential.type,
              response: {
                clientDataJSON: bufferToBase64(credential.response.clientDataJSON),
                authenticatorData: bufferToBase64(credential.response.authenticatorData),
                signature: bufferToBase64(credential.response.signature),
                userHandle: credential.response.userHandle ? bufferToBase64(credential.response.userHandle) : null
              }
            }
          })
        });

        if (!finishRes.ok) {
          const err = await finishRes.json();
          throw new Error(err.error || 'Login failed');
        }

        // Redirect on success
        window.location.href = redirect;
      } catch (err) {
        console.error('Login error:', err);
        if (err.name === 'NotAllowedError') {
          showError(container, 'Authentication cancelled.');
        } else {
          showError(container, err.message || 'Login failed. Please try again.');
        }
      }
    });
  }

  function bindLogout(elem) {
    const redirect = elem.dataset.redirect || '/';

    elem.addEventListener('click', async (e) => {
      e.preventDefault();

      try {
        const res = await fetch('/__auth/logout', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' }
        });

        if (!res.ok) {
          console.error('Logout failed');
        }

        // Redirect regardless of response
        window.location.href = redirect;
      } catch (err) {
        console.error('Logout error:', err);
        window.location.href = redirect;
      }
    });
  }

  function bindAll(root) {
    root.querySelectorAll('form.basil-auth-register:not([data-basil-auth-bound])').forEach((el) => {
      el.setAttribute('data-basil-auth-bound', '');
      bindRegister(el);
    });
    root.querySelectorAll('.basil-auth-login:not([data-basil-auth-bound])').forEach((el) => {
      el.setAttribute('data-basil-auth-bound', '');
      bindLogin(el);
    });
    root.querySelectorAll('.basil-auth-logout:not([data-basil-auth-bound])').forEach((el) => {
      el.setAttribute('data-basil-auth-bound', '');
      bindLogout(el);
    });
  }

  // Classic scripts execute in document order, so elements above the script
  // tag are already parsed; a DOMContentLoaded pass picks up any below it.
  bindAll(document);
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => bindAll(document));
  }
})();
