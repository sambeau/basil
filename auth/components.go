// Package auth provides passkey-based authentication for Basil.
package auth

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// ComponentExpander expands auth component tags in HTML output.
type ComponentExpander struct {
	// tagPatterns maps component names to regex patterns
	tagPatterns map[string]*regexp.Regexp
}

// NewComponentExpander creates a new component expander.
func NewComponentExpander() *ComponentExpander {
	return &ComponentExpander{
		tagPatterns: map[string]*regexp.Regexp{
			"PasskeyRegister": regexp.MustCompile(`<PasskeyRegister\s*([^>]*)/?>`),
			"PasskeyLogin":    regexp.MustCompile(`<PasskeyLogin\s*([^>]*)/?>`),
			"PasskeyLogout":   regexp.MustCompile(`<PasskeyLogout\s*([^>]*)/?>`),
		},
	}
}

// ExpandComponents finds and expands auth component tags in HTML.
func (e *ComponentExpander) ExpandComponents(htmlContent string) string {
	result := htmlContent

	// Expand each component type
	result = e.tagPatterns["PasskeyRegister"].ReplaceAllStringFunc(result, e.expandRegister)
	result = e.tagPatterns["PasskeyLogin"].ReplaceAllStringFunc(result, e.expandLogin)
	result = e.tagPatterns["PasskeyLogout"].ReplaceAllStringFunc(result, e.expandLogout)

	return result
}

// parseAttributes extracts attributes from a tag's attribute string.
// e.g., ` name="Sam" email="sam@example.com"` -> {"name": "Sam", "email": "sam@example.com"}
func parseAttributes(attrStr string) map[string]string {
	attrs := make(map[string]string)

	// Pattern matches: attr_name="value" or attr_name='value'
	attrPattern := regexp.MustCompile(`(\w+)=["']([^"']*)["']`)
	matches := attrPattern.FindAllStringSubmatch(attrStr, -1)

	for _, match := range matches {
		if len(match) == 3 {
			attrs[match[1]] = match[2]
		}
	}

	return attrs
}

// expandRegister expands <PasskeyRegister/> to form HTML + script.
func (e *ComponentExpander) expandRegister(match string) string {
	// Extract attributes
	attrMatch := e.tagPatterns["PasskeyRegister"].FindStringSubmatch(match)
	attrs := parseAttributes(attrMatch[1])

	// Get attribute values with defaults
	name := html.EscapeString(attrs["name"])
	email := html.EscapeString(attrs["email"])
	namePlaceholder := getAttrOrDefault(attrs, "name_placeholder", "Your name")
	emailPlaceholder := getAttrOrDefault(attrs, "email_placeholder", "you@example.com")
	buttonText := getAttrOrDefault(attrs, "button_text", "Create account")
	redirect := getAttrOrDefault(attrs, "redirect", "/")
	class := attrs["class"]

	// Build CSS classes
	formClass := "basil-auth-register"
	if class != "" {
		formClass += " " + html.EscapeString(class)
	}

	// Generate unique ID for this form
	formID := fmt.Sprintf("basil-register-%d", generateUniqueID())

	return fmt.Sprintf(`<form id="%s" class="%s">
  <input type="text" name="name" class="basil-auth-input" placeholder="%s" value="%s" required/>
  <input type="email" name="email" class="basil-auth-input" placeholder="%s" value="%s"/>
  <button type="submit" class="basil-auth-button">%s</button>
  <div class="basil-auth-error" hidden></div>
</form>
<script>
(function() {
  const form = document.getElementById('%s');
  const errorDiv = form.querySelector('.basil-auth-error');
  
  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errorDiv.hidden = true;
    
    const name = form.querySelector('input[name="name"]').value;
    const email = form.querySelector('input[name="email"]').value || null;
    
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
      
      const options = await beginRes.json();
      
      // Step 2: Browser creates credential
      const credential = await navigator.credentials.create({
        publicKey: {
          challenge: base64ToBuffer(options.challenge),
          rp: options.rp,
          user: {
            id: base64ToBuffer(options.user.id),
            name: options.user.name,
            displayName: options.user.displayName
          },
          pubKeyCredParams: options.pubKeyCredParams,
          authenticatorSelection: options.authenticatorSelection,
          timeout: options.timeout,
          attestation: options.attestation
        }
      });
      
      // Step 3: Finish registration
      const finishRes = await fetch('/__auth/register/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: options.session_id,
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
      
      // Show recovery codes if present
      if (result.recovery_codes) {
        alert('Save these recovery codes:\\n\\n' + result.recovery_codes.join('\\n'));
      }
      
      // Redirect on success
      window.location.href = '%s';
      
    } catch (err) {
      console.error('Registration error:', err);
      errorDiv.textContent = err.message || 'Registration failed. Please try again.';
      errorDiv.hidden = false;
    }
  });
  
  // Base64 helpers
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
})();
</script>`,
		formID, formClass, html.EscapeString(namePlaceholder), name,
		html.EscapeString(emailPlaceholder), email,
		html.EscapeString(buttonText), formID, html.EscapeString(redirect))
}

// expandLogin expands <PasskeyLogin/> to button HTML + script.
func (e *ComponentExpander) expandLogin(match string) string {
	// Extract attributes
	attrMatch := e.tagPatterns["PasskeyLogin"].FindStringSubmatch(match)
	attrs := parseAttributes(attrMatch[1])

	// Get attribute values with defaults
	buttonText := getAttrOrDefault(attrs, "button_text", "Sign in")
	redirect := getAttrOrDefault(attrs, "redirect", "/")
	class := attrs["class"]

	// Build CSS classes
	divClass := "basil-auth-login"
	if class != "" {
		divClass += " " + html.EscapeString(class)
	}

	// Generate unique ID for this button
	divID := fmt.Sprintf("basil-login-%d", generateUniqueID())

	return fmt.Sprintf(`<div id="%s" class="%s">
  <button type="button" class="basil-auth-button">%s</button>
  <div class="basil-auth-error" hidden></div>
</div>
<script>
(function() {
  const container = document.getElementById('%s');
  const button = container.querySelector('button');
  const errorDiv = container.querySelector('.basil-auth-error');
  
  button.addEventListener('click', async () => {
    errorDiv.hidden = true;
    
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
      
      const options = await beginRes.json();
      
      // Step 2: Browser authenticates
      const credential = await navigator.credentials.get({
        publicKey: {
          challenge: base64ToBuffer(options.challenge),
          rpId: options.rpId,
          timeout: options.timeout,
          userVerification: options.userVerification,
          allowCredentials: (options.allowCredentials || []).map(c => ({
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
          session_id: options.session_id,
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
      window.location.href = '%s';
      
    } catch (err) {
      console.error('Login error:', err);
      if (err.name === 'NotAllowedError') {
        errorDiv.textContent = 'Authentication cancelled.';
      } else {
        errorDiv.textContent = err.message || 'Login failed. Please try again.';
      }
      errorDiv.hidden = false;
    }
  });
  
  // Base64 helpers
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
})();
</script>`, divID, divClass, html.EscapeString(buttonText), divID, html.EscapeString(redirect))
}

// expandLogout expands <PasskeyLogout/> to button/link HTML + script.
func (e *ComponentExpander) expandLogout(match string) string {
	// Extract attributes
	attrMatch := e.tagPatterns["PasskeyLogout"].FindStringSubmatch(match)
	attrs := parseAttributes(attrMatch[1])

	// Get attribute values with defaults
	text := getAttrOrDefault(attrs, "text", "Sign out")
	redirect := getAttrOrDefault(attrs, "redirect", "/")
	method := getAttrOrDefault(attrs, "method", "button")
	class := attrs["class"]

	// Build CSS classes
	baseClass := "basil-auth-logout"
	if class != "" {
		baseClass += " " + html.EscapeString(class)
	}

	// Generate unique ID
	elemID := fmt.Sprintf("basil-logout-%d", generateUniqueID())

	var element string
	if strings.ToLower(method) == "link" {
		element = fmt.Sprintf(`<a id="%s" href="#" class="%s">%s</a>`,
			elemID, baseClass, html.EscapeString(text))
	} else {
		element = fmt.Sprintf(`<button id="%s" type="button" class="%s basil-auth-button">%s</button>`,
			elemID, baseClass, html.EscapeString(text))
	}

	return fmt.Sprintf(`%s
<script>
(function() {
  const elem = document.getElementById('%s');
  
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
      window.location.href = '%s';
      
    } catch (err) {
      console.error('Logout error:', err);
      window.location.href = '%s';
    }
  });
})();
</script>`, element, elemID, html.EscapeString(redirect), html.EscapeString(redirect))
}

// getAttrOrDefault returns the attribute value or a default.
func getAttrOrDefault(attrs map[string]string, key, defaultVal string) string {
	if val, ok := attrs[key]; ok && val != "" {
		return val
	}
	return defaultVal
}

// uniqueIDCounter is used to generate unique IDs for components.
var uniqueIDCounter int

// generateUniqueID returns a unique integer for component IDs.
func generateUniqueID() int {
	uniqueIDCounter++
	return uniqueIDCounter
}

// ResetUniqueIDCounter resets the counter (for testing).
func ResetUniqueIDCounter() {
	uniqueIDCounter = 0
}
