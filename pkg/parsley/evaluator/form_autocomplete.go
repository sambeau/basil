package evaluator

import "strings"

// autocompletePatterns maps field names (lowercase) to HTML autocomplete values.
// Field names are matched case-insensitively.
var autocompletePatterns = map[string]string{
	// Name fields
	"name":     "name",
	"fullname": "name",
	"full_name": "name",

	"firstname":  "given-name",
	"first_name": "given-name",
	"givenname":  "given-name",
	"given_name": "given-name",
	"forename":   "given-name",

	"lastname":   "family-name",
	"last_name":  "family-name",
	"familyname": "family-name",
	"family_name": "family-name",
	"surname":    "family-name",

	"middlename":      "additional-name",
	"middle_name":     "additional-name",
	"additionalname":  "additional-name",
	"additional_name": "additional-name",

	"nickname": "nickname",
	"nick":     "nickname",

	// Account fields
	"username":  "username",
	"user_name": "username",
	"user":      "username",
	"login":     "username",

	"password": "current-password",
	"passwd":   "current-password",
	"pass":     "current-password",

	"newpassword":      "new-password",
	"new_password":     "new-password",
	"confirmpassword":  "new-password",
	"confirm_password": "new-password",
	"passwordconfirm":  "new-password",
	"password_confirm": "new-password",
	"repeatpassword":   "new-password",
	"repeat_password":  "new-password",

	// Contact fields (in addition to type-based)
	"email":        "email",
	"emailaddress": "email",
	"email_address": "email",
	"mail":         "email",

	"phone":       "tel",
	"phonenumber": "tel",
	"phone_number": "tel",
	"telephone":   "tel",
	"mobile":      "tel",
	"cell":        "tel",

	// Address fields
	"street":        "street-address",
	"streetaddress": "street-address",
	"street_address": "street-address",
	"address":       "street-address",
	"addressline1":  "street-address",
	"address_line_1": "street-address",
	"address1":      "street-address",

	"addressline2":   "address-line2",
	"address_line_2": "address-line2",
	"address2":       "address-line2",
	"apt":            "address-line2",
	"apartment":      "address-line2",
	"suite":          "address-line2",
	"unit":           "address-line2",

	"city":      "address-level2",
	"town":      "address-level2",
	"locality":  "address-level2",

	"state":    "address-level1",
	"province": "address-level1",
	"region":   "address-level1",
	"county":   "address-level1",

	"zip":        "postal-code",
	"zipcode":    "postal-code",
	"zip_code":   "postal-code",
	"postalcode": "postal-code",
	"postal_code": "postal-code",
	"postcode":   "postal-code",
	"post_code":  "postal-code",

	"country":     "country-name",
	"countryname": "country-name",
	"country_name": "country-name",

	"countrycode":  "country",
	"country_code": "country",

	// Organization fields
	"organization": "organization",
	"company":      "organization",
	"org":          "organization",
	"companyname":  "organization",
	"company_name": "organization",
	"employer":     "organization",

	"jobtitle":  "organization-title",
	"job_title": "organization-title",
	"title":     "organization-title",
	"position":  "organization-title",
	"role":      "organization-title",

	// Credit card fields
	"creditcard":      "cc-number",
	"credit_card":     "cc-number",
	"cardnumber":      "cc-number",
	"card_number":     "cc-number",
	"ccnumber":        "cc-number",
	"cc_number":       "cc-number",

	"cardname":    "cc-name",
	"card_name":   "cc-name",
	"ccname":      "cc-name",
	"cc_name":     "cc-name",
	"nameoncard":  "cc-name",
	"name_on_card": "cc-name",

	"cardexpiry":       "cc-exp",
	"card_expiry":      "cc-exp",
	"ccexpiry":         "cc-exp",
	"cc_expiry":        "cc-exp",
	"expiry":           "cc-exp",
	"expirationdate":   "cc-exp",
	"expiration_date":  "cc-exp",
	"cardexpiration":   "cc-exp",
	"card_expiration":  "cc-exp",

	"cardcvc":       "cc-csc",
	"card_cvc":      "cc-csc",
	"cvc":           "cc-csc",
	"cvv":           "cc-csc",
	"securitycode":  "cc-csc",
	"security_code": "cc-csc",
	"csc":           "cc-csc",

	// Other fields
	"birthday":     "bday",
	"birthdate":    "bday",
	"birth_date":   "bday",
	"dob":          "bday",
	"dateofbirth":  "bday",
	"date_of_birth": "bday",

	"language":          "language",
	"preferredlanguage": "language",
	"preferred_language": "language",

	"otp":              "one-time-code",
	"verificationcode": "one-time-code",
	"verification_code": "one-time-code",
	"code":             "one-time-code",
}

// typeAutocomplete maps schema types to HTML autocomplete values.
var typeAutocomplete = map[string]string{
	"email": "email",
	"phone": "tel",
	"tel":   "tel",
	"url":   "url",
}

// getAutocomplete returns the HTML autocomplete attribute value for a field.
// Priority: explicit metadata > field name pattern > schema type > empty.
// Returns empty string if no autocomplete should be added.
func getAutocomplete(fieldName, fieldType string, metadata map[string]Object) string {
	// 1. Check explicit metadata (always wins)
	if metadata != nil {
		if ac, ok := metadata["autocomplete"]; ok {
			if strVal, ok := ac.(*String); ok {
				return strVal.Value
			}
			// If it's some other type, convert to string
			return objectToTemplateString(ac)
		}
	}

	// 2. Check field name patterns (case-insensitive)
	nameLower := strings.ToLower(fieldName)
	if ac, ok := autocompletePatterns[nameLower]; ok {
		return ac
	}

	// 3. Check type-based defaults
	baseType := strings.TrimSuffix(strings.ToLower(fieldType), "?") // Strip nullable marker
	if ac, ok := typeAutocomplete[baseType]; ok {
		return ac
	}

	// 4. No match - return empty (no autocomplete attribute)
	return ""
}
