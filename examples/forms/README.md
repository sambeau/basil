# Form Binding Examples

This directory contains examples demonstrating Record-based form binding in Basil.

## Overview

These examples showcase:

1. **Basic form binding** - Using `@record` and `@field` attributes
2. **Validation errors** - Displaying validation errors inline
3. **Database integration** - Loading records from DB for editing
4. **Complete CRUD flow** - Full create/read/update workflow

## Running Examples

Start the Basil server:

```bash
./basil examples/forms
```

Then visit `http://localhost:3000` in your browser.

## Key Concepts

### Schema Definition

```parsley
@schema Contact {
    name: string(title: "Full Name", min: 2)
    email: email(title: "Email Address")
    message: string(title: "Message", placeholder: "Enter your message...")
}
```

### Form Binding

```parsley
let contact = Contact({...})

<form @record={contact} method="post">
    <input @field="name"/>
    <input @field="email"/>
    <textarea @field="message"/>
    <button type="submit">Submit</button>
</form>
```

### Error Display

```parsley
{if contact.hasError("email") then
    <span class="error">{contact.error("email")}</span>
}
```

## Files

| File | Description |
|------|-------------|
| `handlers/contact-form.pars` | Basic contact form with validation |
| `handlers/edit-user.pars` | Edit existing record from database |
| `handlers/user-list.pars` | List users with edit links |
| `handlers/form.part` | Interactive form Part with validation |
