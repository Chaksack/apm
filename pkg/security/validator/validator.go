package validator

import (
	"fmt"
	"html"
	"reflect"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

// Validator handles input validation
type Validator struct {
	logger *zap.Logger
}

// NewValidator creates a new validator
func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// ValidateStruct validates a struct against rules
func (v *Validator) ValidateStruct(data interface{}, rules map[string][]ValidationRule) error {
	var errors ValidationErrors

	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() == reflect.Ptr {
		dataValue = dataValue.Elem()
	}

	dataType := dataValue.Type()

	for fieldName, fieldRules := range rules {
		// Get field value
		fieldValue := dataValue.FieldByName(fieldName)
		if !fieldValue.IsValid() {
			// Try lowercase field name
			for i := 0; i < dataType.NumField(); i++ {
				field := dataType.Field(i)
				jsonTag := field.Tag.Get("json")
				if jsonTag == fieldName {
					fieldValue = dataValue.Field(i)
					break
				}
			}
		}

		if !fieldValue.IsValid() {
			v.logger.Debug("field not found", zap.String("field", fieldName))
			continue
		}

		// Validate field
		for _, rule := range fieldRules {
			if err := v.validateField(fieldName, fieldValue.Interface(), rule); err != nil {
				errors = append(errors, err...)
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// ValidateMap validates a map against rules
func (v *Validator) ValidateMap(data map[string]interface{}, rules map[string][]ValidationRule) error {
	var errors ValidationErrors

	for fieldName, fieldRules := range rules {
		value, exists := data[fieldName]

		// Check required fields
		if !exists {
			for _, rule := range fieldRules {
				if rule.Required {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: "is required",
					})
					break
				}
			}
			continue
		}

		// Validate field
		for _, rule := range fieldRules {
			if err := v.validateField(fieldName, value, rule); err != nil {
				errors = append(errors, err...)
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateField validates a single field against a rule
func (v *Validator) validateField(fieldName string, value interface{}, rule ValidationRule) ValidationErrors {
	var errors ValidationErrors

	// Handle nil/empty values
	if value == nil || isZeroValue(value) {
		if rule.Required {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "is required",
			})
		}
		return errors
	}

	// Convert to string for string validations
	strValue := fmt.Sprintf("%v", value)

	// Min length validation
	if rule.MinLength > 0 && len(strValue) < rule.MinLength {
		errors = append(errors, ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("must be at least %d characters", rule.MinLength),
		})
	}

	// Max length validation
	if rule.MaxLength > 0 && len(strValue) > rule.MaxLength {
		errors = append(errors, ValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("must not exceed %d characters", rule.MaxLength),
		})
	}

	// Pattern validation
	if rule.Pattern != "" {
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			v.logger.Error("invalid pattern", zap.String("pattern", rule.Pattern), zap.Error(err))
		} else if !pattern.MatchString(strValue) {
			message := rule.PatternDesc
			if message == "" {
				message = fmt.Sprintf("must match pattern %s", rule.Pattern)
			}
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: message,
			})
		}
	}

	// Numeric validations
	if rule.MinValue != nil || rule.MaxValue != nil {
		numValue, ok := toFloat64(value)
		if !ok {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: "must be a number",
			})
		} else {
			if rule.MinValue != nil && numValue < *rule.MinValue {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must be at least %v", *rule.MinValue),
				})
			}
			if rule.MaxValue != nil && numValue > *rule.MaxValue {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: fmt.Sprintf("must not exceed %v", *rule.MaxValue),
				})
			}
		}
	}

	// Enum validation
	if len(rule.Enum) > 0 {
		valid := false
		for _, enumValue := range rule.Enum {
			if strValue == enumValue {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, ValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("must be one of: %s", strings.Join(rule.Enum, ", ")),
			})
		}
	}

	// Custom validation
	if rule.Custom != nil {
		if err := rule.Custom(value); err != nil {
			if validationErr, ok := err.(ValidationErrors); ok {
				errors = append(errors, validationErr...)
			} else {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Message: err.Error(),
				})
			}
		}
	}

	return errors
}

// Sanitizer handles input sanitization
type Sanitizer struct {
	logger *zap.Logger
}

// NewSanitizer creates a new sanitizer
func NewSanitizer(logger *zap.Logger) *Sanitizer {
	return &Sanitizer{
		logger: logger,
	}
}

// SanitizeString sanitizes a string value
func (s *Sanitizer) SanitizeString(value string, rules []SanitizationRule) string {
	result := value

	for _, rule := range rules {
		// Trim space
		if rule.TrimSpace {
			result = strings.TrimSpace(result)
		}

		// Case conversion
		if rule.ToLower {
			result = strings.ToLower(result)
		}
		if rule.ToUpper {
			result = strings.ToUpper(result)
		}

		// Remove HTML
		if rule.RemoveHTML {
			result = html.EscapeString(result)
		}

		// Remove script tags
		if rule.RemoveScript {
			scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
			result = scriptPattern.ReplaceAllString(result, "")
		}

		// Max length
		if rule.MaxLength > 0 && len(result) > rule.MaxLength {
			result = result[:rule.MaxLength]
		}

		// Allowed characters
		if rule.AllowedChars != "" {
			allowedPattern := regexp.MustCompile(fmt.Sprintf("[^%s]", regexp.QuoteMeta(rule.AllowedChars)))
			result = allowedPattern.ReplaceAllString(result, "")
		}

		// Replace patterns
		for pattern, replacement := range rule.ReplacePatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				s.logger.Error("invalid replacement pattern", zap.String("pattern", pattern), zap.Error(err))
				continue
			}
			result = re.ReplaceAllString(result, replacement)
		}

		// Custom sanitization
		if rule.Custom != nil {
			result = rule.Custom(result)
		}
	}

	return result
}

// SanitizeMap sanitizes a map of values
func (s *Sanitizer) SanitizeMap(data map[string]interface{}, rules map[string][]SanitizationRule) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy all values
	for k, v := range data {
		result[k] = v
	}

	// Apply sanitization rules
	for fieldName, fieldRules := range rules {
		if value, exists := result[fieldName]; exists {
			if strValue, ok := value.(string); ok {
				result[fieldName] = s.SanitizeString(strValue, fieldRules)
			}
		}
	}

	return result
}

// Helper functions

// isZeroValue checks if a value is zero/empty
func isZeroValue(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	}

	return false
}

// toFloat64 converts a value to float64
func toFloat64(value interface{}) (float64, bool) {
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Float(), true
	}

	return 0, false
}
