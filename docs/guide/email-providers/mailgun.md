# Mailgun Setup Guide

Mailgun is a transactional email service with excellent deliverability and a generous free tier.

## Prerequisites

- A domain you control
- Mailgun account (free or paid)

## Step 1: Create Mailgun Account

1. Go to [mailgun.com](https://www.mailgun.com/)
2. Sign up for free account (3 months free, then pay-as-you-go)
3. Verify your email address

## Step 2: Add and Verify Domain

### Option A: Use Sandbox Domain (Testing Only)

Mailgun provides a sandbox domain immediately:

```
sandbox1234567890abcdef.mailgun.org
```

**Limitations:**
- Can only send to authorized recipients
- Limited to 300 emails/month
- Emails marked as "via mailgun.org"
- Not suitable for production

**Configuration:**
```yaml
email_verification:
  enabled: true
  provider: mailgun
  mailgun:
    api_key: "your-api-key"
    domain: "sandbox1234567890abcdef.mailgun.org"
    from: "noreply@sandbox1234567890abcdef.mailgun.org"
```

### Option B: Add Custom Domain (Production)

1. Go to **Sending** → **Domains**
2. Click **Add New Domain**
3. Enter your domain (e.g., `mg.example.com` or `example.com`)
4. Choose region: US or EU
5. Click **Add Domain**

### Option C: Use Subdomain (Recommended)

Best practice is to use a subdomain:

**Advantages:**
- Doesn't affect your main domain's email reputation
- Isolates transactional email from marketing email
- Easier to manage DNS records

**Example:** `mg.example.com` or `mail.example.com`

## Step 3: Configure DNS Records

Mailgun requires several DNS records for domain verification:

### Required Records

Add these DNS records to your domain:

**TXT Records (Domain Verification):**
```
Name: @ (or your subdomain)
Type: TXT
Value: v=spf1 include:mailgun.org ~all
```

**TXT Record (DKIM):**
```
Name: k1._domainkey (or subdomain.k1._domainkey)
Type: TXT
Value: k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA4... (provided by Mailgun)
```

**CNAME Records (Tracking):**
```
Name: email (or subdomain.email)
Type: CNAME
Value: mailgun.org
```

### Verification

1. Add all DNS records to your DNS provider
2. Wait 5-15 minutes for propagation
3. Return to Mailgun dashboard
4. Click **Verify DNS Settings**
5. All records should show green checkmarks

**Tip:** Use `dig` or `nslookup` to verify DNS propagation:

```bash
dig TXT mg.example.com
dig TXT k1._domainkey.mg.example.com
```

## Step 4: Get API Key

1. Go to **Settings** → **API Keys**
2. Copy your **Private API Key** (starts with `key-`)
3. Store securely (do not commit to git)

**Security:** API keys grant full account access. Keep them secret!

## Step 5: Configure Basil

Create or update `basil.yaml`:

```yaml
auth:
  enabled: true
  
  email_verification:
    enabled: true
    provider: mailgun
    
    mailgun:
      api_key: "${MAILGUN_API_KEY}"
      domain: "mg.example.com"
      region: us  # or "eu" for Europe
      from: "noreply@mg.example.com"
    
    require_verification: true
    token_ttl: 1h
    resend_cooldown: 5m
    max_sends_per_day: 10
```

Set environment variable:

```bash
export MAILGUN_API_KEY="key-your-mailgun-api-key"
```

## Step 6: Test Email Sending

### Using Development Credentials

Basil provides test credentials for development:

```yaml
mailgun:
  api_key: "8d62a15e184a96921189f2976fae04cd-f6d80573-cb1a0b2a"
  domain: "mg.tickly.org"
  from: "test@mg.tickly.org"
```

Send test email to `sambeau@mac.com`

### Test with Your Domain

1. Start Basil: `basil`
2. Register a new account
3. Check your email for verification link
4. Click link to verify

### Check Mailgun Logs

1. Go to **Sending** → **Logs**
2. View delivery status, opens, clicks
3. Debug any delivery issues

## Region Configuration

Mailgun has two regions:

### US Region (Default)

```yaml
mailgun:
  region: us
```

API endpoint: `https://api.mailgun.net/v3`

Use for:
- US-based users
- Global users (default)

### EU Region

```yaml
mailgun:
  region: eu
```

API endpoint: `https://api.eu.mailgun.net/v3`

Use for:
- GDPR compliance
- EU-based users
- Data sovereignty requirements

## Pricing

**Free Tier:**
- First 3 months free
- Up to 5,000 emails/month

**Pay-as-you-go:**
- $0.80 per 1,000 emails
- No monthly minimum
- Volume discounts available

**Foundation Plan:**
- $35/month
- 50,000 emails included
- Additional emails: $0.80/1,000

[Full pricing](https://www.mailgun.com/pricing/)

## Deliverability Best Practices

### 1. Warm Up Your Domain

Start with low volumes and gradually increase:
- Week 1: 50-100 emails/day
- Week 2: 200-500 emails/day
- Week 3: 1,000+ emails/day

### 2. Monitor Metrics

Check Mailgun dashboard for:
- Delivery rate (aim for >95%)
- Bounce rate (keep <5%)
- Complaint rate (keep <0.1%)

### 3. Handle Bounces

Implement bounce handling:
- Hard bounces: Remove from list
- Soft bounces: Retry automatically
- Complaints: Unsubscribe immediately

### 4. Authenticate Email

Required DNS records improve deliverability:
- SPF: Prevents spoofing
- DKIM: Digital signature
- DMARC: Policy enforcement

### 5. Maintain Clean List

- Use double opt-in (email verification does this)
- Remove invalid addresses
- Monitor engagement

## Troubleshooting

### Domain Not Verifying

**Problem:** DNS records not showing as verified

**Solutions:**
1. Wait 15-30 minutes for DNS propagation
2. Check DNS records with `dig` or DNS checker
3. Verify you added records to correct DNS provider
4. Remove and re-add records if needed

### Emails Going to Spam

**Problem:** Verification emails landing in spam folder

**Solutions:**
1. Complete domain verification (SPF, DKIM)
2. Add DMARC record
3. Warm up domain gradually
4. Improve email content (avoid spam trigger words)
5. Ask recipients to whitelist your domain

### API Key Not Working

**Problem:** Authentication errors

**Solutions:**
1. Verify API key starts with `key-`
2. Check for extra spaces/newlines
3. Ensure environment variable is set
4. Regenerate API key if needed

### Rate Limit Errors

**Problem:** Mailgun returning rate limit errors

**Solutions:**
1. Upgrade to paid plan
2. Implement exponential backoff
3. Spread sends over time
4. Contact Mailgun support for limit increase

## Support Resources

- [Mailgun Documentation](https://documentation.mailgun.com/)
- [API Reference](https://documentation.mailgun.com/en/latest/api_reference.html)
- [Support](https://help.mailgun.com/)
- [Status Page](https://status.mailgun.com/)

## See Also

- [Email Verification Guide](../email-verification.md)
- [Resend Setup Guide](./resend.md)
- [Authentication Guide](../authentication.md)
