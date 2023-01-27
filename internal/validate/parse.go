package validate

import "strings"

const (
	allowedRegistriesSeparator = ","
)

func ParseAllowedRegistries(registries string) []string {
	var registriesList []string
	for _, registry := range strings.Split(registries, allowedRegistriesSeparator) {
		sanitizedRegistry := strings.TrimSpace(registry)
		if sanitizedRegistry != "" {
			registriesList = append(registriesList, sanitizedRegistry)
		}
	}

	return registriesList
}
