package evaluator

import "testing"

func TestGetAutocomplete(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		fieldType string
		metadata  map[string]Object
		want      string
	}{
		// Type-based defaults
		{
			name:      "email type derives email",
			fieldName: "contactEmail",
			fieldType: "email",
			metadata:  nil,
			want:      "email",
		},
		{
			name:      "phone type derives tel",
			fieldName: "homePhone",
			fieldType: "phone",
			metadata:  nil,
			want:      "tel",
		},
		{
			name:      "tel type derives tel",
			fieldName: "mobile",
			fieldType: "tel",
			metadata:  nil,
			want:      "tel",
		},
		{
			name:      "url type derives url",
			fieldName: "website",
			fieldType: "url",
			metadata:  nil,
			want:      "url",
		},
		{
			name:      "nullable email type derives email",
			fieldName: "secondaryEmail",
			fieldType: "email?",
			metadata:  nil,
			want:      "email",
		},

		// Field name patterns - names
		{
			name:      "firstName derives given-name",
			fieldName: "firstName",
			fieldType: "string",
			metadata:  nil,
			want:      "given-name",
		},
		{
			name:      "FIRSTNAME case-insensitive",
			fieldName: "FIRSTNAME",
			fieldType: "string",
			metadata:  nil,
			want:      "given-name",
		},
		{
			name:      "FirstName mixed case",
			fieldName: "FirstName",
			fieldType: "string",
			metadata:  nil,
			want:      "given-name",
		},
		{
			name:      "first_name snake_case",
			fieldName: "first_name",
			fieldType: "string",
			metadata:  nil,
			want:      "given-name",
		},
		{
			name:      "lastName derives family-name",
			fieldName: "lastName",
			fieldType: "string",
			metadata:  nil,
			want:      "family-name",
		},
		{
			name:      "surname derives family-name",
			fieldName: "surname",
			fieldType: "string",
			metadata:  nil,
			want:      "family-name",
		},
		{
			name:      "fullName derives name",
			fieldName: "fullName",
			fieldType: "string",
			metadata:  nil,
			want:      "name",
		},

		// Field name patterns - account
		{
			name:      "username derives username",
			fieldName: "username",
			fieldType: "string",
			metadata:  nil,
			want:      "username",
		},
		{
			name:      "password derives current-password",
			fieldName: "password",
			fieldType: "string",
			metadata:  nil,
			want:      "current-password",
		},
		{
			name:      "newPassword derives new-password",
			fieldName: "newPassword",
			fieldType: "string",
			metadata:  nil,
			want:      "new-password",
		},
		{
			name:      "confirmPassword derives new-password",
			fieldName: "confirmPassword",
			fieldType: "string",
			metadata:  nil,
			want:      "new-password",
		},

		// Field name patterns - address
		{
			name:      "street derives street-address",
			fieldName: "street",
			fieldType: "string",
			metadata:  nil,
			want:      "street-address",
		},
		{
			name:      "city derives address-level2",
			fieldName: "city",
			fieldType: "string",
			metadata:  nil,
			want:      "address-level2",
		},
		{
			name:      "state derives address-level1",
			fieldName: "state",
			fieldType: "string",
			metadata:  nil,
			want:      "address-level1",
		},
		{
			name:      "zipCode derives postal-code",
			fieldName: "zipCode",
			fieldType: "string",
			metadata:  nil,
			want:      "postal-code",
		},
		{
			name:      "country derives country-name",
			fieldName: "country",
			fieldType: "string",
			metadata:  nil,
			want:      "country-name",
		},

		// Field name patterns - organization
		{
			name:      "organization derives organization",
			fieldName: "organization",
			fieldType: "string",
			metadata:  nil,
			want:      "organization",
		},
		{
			name:      "company derives organization",
			fieldName: "company",
			fieldType: "string",
			metadata:  nil,
			want:      "organization",
		},
		{
			name:      "jobTitle derives organization-title",
			fieldName: "jobTitle",
			fieldType: "string",
			metadata:  nil,
			want:      "organization-title",
		},

		// Field name patterns - credit card
		{
			name:      "cardNumber derives cc-number",
			fieldName: "cardNumber",
			fieldType: "string",
			metadata:  nil,
			want:      "cc-number",
		},
		{
			name:      "cardName derives cc-name",
			fieldName: "cardName",
			fieldType: "string",
			metadata:  nil,
			want:      "cc-name",
		},
		{
			name:      "cardExpiry derives cc-exp",
			fieldName: "cardExpiry",
			fieldType: "string",
			metadata:  nil,
			want:      "cc-exp",
		},
		{
			name:      "cvv derives cc-csc",
			fieldName: "cvv",
			fieldType: "string",
			metadata:  nil,
			want:      "cc-csc",
		},

		// Field name patterns - other
		{
			name:      "birthday derives bday",
			fieldName: "birthday",
			fieldType: "string",
			metadata:  nil,
			want:      "bday",
		},
		{
			name:      "dob derives bday",
			fieldName: "dob",
			fieldType: "string",
			metadata:  nil,
			want:      "bday",
		},

		// Explicit metadata override
		{
			name:      "explicit metadata overrides type",
			fieldName: "email",
			fieldType: "email",
			metadata:  map[string]Object{"autocomplete": &String{Value: "off"}},
			want:      "off",
		},
		{
			name:      "explicit metadata overrides field name",
			fieldName: "password",
			fieldType: "string",
			metadata:  map[string]Object{"autocomplete": &String{Value: "new-password"}},
			want:      "new-password",
		},
		{
			name:      "explicit shipping compound value",
			fieldName: "shippingStreet",
			fieldType: "string",
			metadata:  map[string]Object{"autocomplete": &String{Value: "shipping street-address"}},
			want:      "shipping street-address",
		},
		{
			name:      "explicit billing compound value",
			fieldName: "billingCity",
			fieldType: "string",
			metadata:  map[string]Object{"autocomplete": &String{Value: "billing address-level2"}},
			want:      "billing address-level2",
		},

		// No match cases
		{
			name:      "unknown field returns empty",
			fieldName: "favoriteColor",
			fieldType: "string",
			metadata:  nil,
			want:      "",
		},
		{
			name:      "unknown type returns empty",
			fieldName: "data",
			fieldType: "json",
			metadata:  nil,
			want:      "",
		},
		{
			name:      "empty metadata returns field name match",
			fieldName: "email",
			fieldType: "string",
			metadata:  map[string]Object{},
			want:      "email",
		},
		{
			name:      "nil metadata uses type and field name",
			fieldName: "phone",
			fieldType: "phone",
			metadata:  nil,
			want:      "tel",
		},

		// Priority: metadata > field name > type
		{
			name:      "field name takes priority over type for email named password",
			fieldName: "password",
			fieldType: "email", // weird but tests priority
			metadata:  nil,
			want:      "current-password",
		},
		{
			name:      "metadata takes priority over everything",
			fieldName: "email",
			fieldType: "email",
			metadata:  map[string]Object{"autocomplete": &String{Value: "username"}},
			want:      "username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAutocomplete(tt.fieldName, tt.fieldType, tt.metadata)
			if got != tt.want {
				t.Errorf("getAutocomplete(%q, %q, %v) = %q, want %q",
					tt.fieldName, tt.fieldType, tt.metadata, got, tt.want)
			}
		})
	}
}

func TestAutocompletePatternCoverage(t *testing.T) {
	// Verify all documented patterns are present
	expectedPatterns := map[string]string{
		// Names
		"name":            "name",
		"fullname":        "name",
		"firstname":       "given-name",
		"givenname":       "given-name",
		"lastname":        "family-name",
		"familyname":      "family-name",
		"surname":         "family-name",
		"middlename":      "additional-name",
		"nickname":        "nickname",

		// Account
		"username":        "username",
		"password":        "current-password",
		"newpassword":     "new-password",
		"confirmpassword": "new-password",

		// Contact
		"email":           "email",
		"phone":           "tel",
		"mobile":          "tel",

		// Address
		"street":          "street-address",
		"streetaddress":   "street-address",
		"address":         "street-address",
		"addressline2":    "address-line2",
		"city":            "address-level2",
		"state":           "address-level1",
		"province":        "address-level1",
		"zip":             "postal-code",
		"zipcode":         "postal-code",
		"postalcode":      "postal-code",
		"country":         "country-name",
		"countrycode":     "country",

		// Organization
		"organization":    "organization",
		"company":         "organization",
		"jobtitle":        "organization-title",
		"title":           "organization-title",

		// Credit card
		"cardnumber":      "cc-number",
		"creditcard":      "cc-number",
		"cardname":        "cc-name",
		"cardexpiry":      "cc-exp",
		"cvv":             "cc-csc",
		"cvc":             "cc-csc",

		// Other
		"birthday":        "bday",
		"dob":             "bday",
		"language":        "language",
		"otp":             "one-time-code",
		"verificationcode": "one-time-code",
	}

	for pattern, expected := range expectedPatterns {
		if got, ok := autocompletePatterns[pattern]; !ok {
			t.Errorf("Pattern %q not found in autocompletePatterns", pattern)
		} else if got != expected {
			t.Errorf("autocompletePatterns[%q] = %q, want %q", pattern, got, expected)
		}
	}
}

func TestTypeAutocompleteCoverage(t *testing.T) {
	// Verify type-based defaults
	expectedTypes := map[string]string{
		"email": "email",
		"phone": "tel",
		"tel":   "tel",
		"url":   "url",
	}

	for typ, expected := range expectedTypes {
		if got, ok := typeAutocomplete[typ]; !ok {
			t.Errorf("Type %q not found in typeAutocomplete", typ)
		} else if got != expected {
			t.Errorf("typeAutocomplete[%q] = %q, want %q", typ, got, expected)
		}
	}
}
