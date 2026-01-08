# Resend Setup Guide

Resend is a modern email API built for developers, with a focus on simplicity and great developer experience.

## Prerequisites

- A domain you control (optional for testing)
- Resend account (free tier available)

## Step 1: Create Resend Account

1. Go to [resend.com](https://resend.com/)
2. Sign up with GitHub, Google, or email
3. Verify your email address

## Step 2: Get API Key

1. Go to **API Keys** in dashboard
2. Click **Create API Key**
3. Name it (e.g., "Basil Production")
4. Choose permissions: **Sending access**
5. Click **Create**
6. Copy the API key (starts with `re_`)

**Important:** API key is shown only once. Store securely.

## Step 3: Domain Setup

### Option A: Use Onboarding Email (Testing Only)

Resend provides immediate testing email:

```
from: "onboarding@resend.dev"
```

**Limitations:**
- Only for testing and development
- Limited deliverability
- Branded as Resend
- Not for production use

**Configuration:**
```yaml
email_verification:
  enabled: true
  provider: resend
  resend:
    api_key: "${RESEND_API_KEY}"
    from: "onboarding@resend.dev"
```

**No domain setup required!** Start testing immediately.

### Option B: Add Custom Domain (Production)

For production, add your own domain:

1. Go to **Domains** in dashboard
2. Click **Add Domain**
3. Enter your domain (e.g., `example.com`)
4. Click **Add**

**Subdomain Recommended:** Use `mail.example.com` or similar to isolate transactional email.

## Step 4: Verify Domain

### DNS Records

Resend requires DNS verification:

**TXT Record (Verification):**
```
Name: @ (or subdomain)
Type: TXT
Value: resend-verify=<verification-token>
```

**SPF Record (Sending):**
```
Name: @ (or subdomain)
Type: TXT
Value: v=spf1 include:resend.com ~all
```

**DKIM Records (Authentication):**

Resend provides three DKIM records:

```
Name: resend._domainkey
Type: TXT
Value: <dkim-public-key>

Name: resend._domainkey.example.com
Type: CNAME
Value: resend.com
```

### Add Records to DNS

1. Log into your DNS provider (Cloudflare, Route53, etc.)
2. Add all records provided by Resend
3. Wait 5-15 minutes for propagation
4. Return to Resend dashboard
5. Click **Verify Records**

### Verification Status

Green checkmarks indicate success:
- ✅ Domain ownership verified
- ✅ SPF configured
- ✅ DKIM configured

## Step 5: Configure Basil

Update `basil.yaml`:

```yaml
auth:
  enabled: true
  
  email_verification:
    enabled: true
    provider: resend
    
    resend:
      api_key: "${RESEND_API_KEY}"
      from: "noreply@example.com"
    
    require_verification: true
    token_ttl: 1h
    resend_cooldown: 5m
    max_sends_per_day: 10
    
    template_vars:
      site_name: "My App"
      site_url: "https://example.com"
```

Set environment variable:

```bash
export RESEND_API_KEY="re_your-api-key"
```

## Step 6: Test Email Sending

### Using Development Email

Test immediately with onboarding email:

```yaml
resend:
  api_key: "re_CTmuvKDZ_Hs8QFRGe2hnRQWmPt3uq9nxJ"
  from: "onboarding@resend.dev"
```

Send test to: `sambeau@mac.com`

### Test with Your Domain

1. Start Basil: `basil`
2. Register new account with your email
3. Check inbox for verification email
4. Click verification link

### Check Resend Logs

1. Go to **Emails** in dashboard
2. View sent emails, delivery status
3. Debug any issues

## From Address Format

The `from` field can be:

**Simple email:**
```yaml
from: "noreply@example.com"
```

**With display name:**
```yaml
from: "My App <noreply@example.com>"
```

**Best practice:** Use descriptive name so users recognize your app:
```yaml
from: "My App Verification <verify@example.com>"
```

## Pricing

**Free Tier:**
- 3,000 emails/month
- All features included
- 1 verified domain
- No credit card required

**Pro Plan:**
- $20/month
- 50,000 emails/month
- Additional: $1 per 1,000 emails
- 10 verified domains
- Priority support

**Scale Plan:**
- Custom pricing
- Unlimited domains
- Dedicated IPs
- Custom SLAs

[Full pricing](https://resend.com/pricing)

## Features

### Email Testing

Resend provides test emails that don't hit real inboxes:

```bash
curl -X POST https://api.resend.com/emails \
  -H "Authorization: Bearer $RESEND_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"to": "delivered@resend.dev", ...}'
```

Special addresses:
- `delivered@resend.dev` - Simulates successful delivery
- `bounced@resend.dev` - Simulates bounce
- `complained@resend.dev` - Simulates complaint

### React Email Integration

Resend integrates with React Email for beautiful templates:

```bash
npm install react-email @react-email/components
```

(Not used by Basil currently - we use Go templates)

### Webhooks

Configure webhooks to track:
- Email delivered
- Email opened
- Email clicked
- Email bounced
- Email complained

## Deliverability Best Practices

### 1. Verify Domain Properly

Ensure all DNS records are configured:
- ✅ Domain verification
- ✅ SPF record
- ✅ DKIM records
- ✅ Optional: DMARC

### 2. Use Recognizable From Address

Users are more likely to trust:
```
"Your App Name <noreply@yourapp.com>"
```

Than:
```
"noreply@example.com"
```

### 3. Write Clear Subject Lines

Verification email subjects should be clear:
- ✅ "Verify your email for My App"
- ❌ "Action required!!!"

### 4. Monitor Engagement

Check Resend dashboard for:
- Delivery rate (aim for >98%)
- Open rate
- Click rate
- Bounce rate (keep <2%)

### 5. Handle Bounces

Implement bounce handling:
- Remove hard bounces immediately
- Retry soft bounces
- Monitor complaint rate

## Troubleshooting

### API Key Not Working

**Problem:** Authentication errors

**Solutions:**
1. Verify API key starts with `re_`
2. Check environment variable is set correctly
3. Ensure no trailing spaces/newlines
4. Regenerate API key if compromised

### Domain Not Verifying

**Problem:** DNS records not verified

**Solutions:**
1. Wait 10-20 minutes for DNS propagation
2. Use `dig` to verify records are live:
   ```bash
   dig TXT example.com
   dig TXT resend._domainkey.example.com
   ```
3. Check records are added to correct DNS zone
4. Remove and re-add records if needed

### Emails Not Arriving

**Problem:** Users not receiving verification emails

**Solutions:**
1. Check spam folder
2. Verify domain is fully verified in Resend
3. Check Resend dashboard for delivery status
4. Ensure email address is valid
5. Check rate limits haven't been exceeded

### Rate Limit Errors

**Problem:** Hitting free tier limits

**Solutions:**
1. Upgrade to Pro plan ($20/month for 50k emails)
2. Implement rate limiting in application
3. Use Basil's built-in rate limits
4. Contact Resend support

### From Address Rejected

**Problem:** Resend rejecting from address

**Solutions:**
1. Ensure domain is verified in Resend
2. Use email address on verified domain
3. Check from address format is valid
4. Don't use `@resend.dev` addresses (except onboarding)

## Comparing Resend vs Mailgun

| Feature | Resend | Mailgun |
|---------|--------|---------|
| Free tier | 3,000/month | 5,000/month (3 months) |
| Setup complexity | Simple | Moderate |
| Dashboard UX | Modern | Traditional |
| API design | Developer-friendly | Comprehensive |
| Testing | Built-in test addresses | Sandbox domains |
| Pricing | $20/month (50k) | $35/month (50k) |
| Best for | Startups, developers | Enterprise, scale |

**Choose Resend if:**
- You want quick setup
- You prefer modern developer UX
- You're building a startup/side project

**Choose Mailgun if:**
- You need enterprise features
- You send very high volumes
- You need regional data compliance

## Support Resources

- [Resend Documentation](https://resend.com/docs)
- [API Reference](https://resend.com/docs/api-reference)
- [Discord Community](https://resend.com/discord)
- [GitHub Examples](https://github.com/resendlabs/resend-examples)
- [Status Page](https://status.resend.com/)

## See Also

- [Email Verification Guide](../email-verification.md)
- [Mailgun Setup Guide](./mailgun.md)
- [Authentication Guide](../authentication.md)
