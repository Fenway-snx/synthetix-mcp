package halt

func ValidateRequiredMetadata(adminAPIKey, serviceId, version string) []string {
	validationErrors := make([]string, 0, 3)

	if adminAPIKey == "" {
		validationErrors = append(validationErrors, "admin_api_key is required for admin operations")
	}
	if serviceId == "" {
		validationErrors = append(validationErrors, "service_id is required for admin state reporting")
	}
	if version == "" {
		validationErrors = append(validationErrors, "version is required for admin state reporting")
	}

	return validationErrors
}
