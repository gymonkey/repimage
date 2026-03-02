package utils

import (
	"strings"
)

const (
	defaultDomain    = "docker.io"
	officialRepoName = "library"
)

// ReplaceImageName adds a mirror prefix to container image names, but skips domains in ignoreDomains
func ReplaceImageName(prefix string, ignoreDomains []string, name string) string {
	parts := strings.SplitN(name, "/", 3)
	if parts[0] == prefix {
		return name
	}

	switch len(parts) {
	case 1:
		if shouldIgnoreDomain(defaultDomain, ignoreDomains) {
			return strings.Join([]string{defaultDomain, officialRepoName, parts[0]}, "/")
		}

		return strings.Join([]string{prefix, defaultDomain, officialRepoName, parts[0]}, "/")
	case 2:
		if !isDomain(parts[0]) {
			if shouldIgnoreDomain(defaultDomain, ignoreDomains) {
				return strings.Join([]string{defaultDomain, parts[0], parts[1]}, "/")
			}

			return strings.Join([]string{prefix, defaultDomain, parts[0], parts[1]}, "/")
		}

		if isLegacyDefaultDomain(parts[0]) {
			parts[0] = defaultDomain
		}

		if shouldIgnoreDomain(parts[0], ignoreDomains) {
			return strings.Join([]string{parts[0], parts[1]}, "/")
		}

		return strings.Join([]string{prefix, parts[0], parts[1]}, "/")
	case 3:
		if !isDomain(parts[0]) {
			if shouldIgnoreDomain(defaultDomain, ignoreDomains) {
				return strings.Join([]string{defaultDomain, parts[0], parts[1], parts[2]}, "/")
			}

			return strings.Join([]string{prefix, defaultDomain, parts[0], parts[1], parts[2]}, "/")
		}

		if isLegacyDefaultDomain(parts[0]) {
			parts[0] = defaultDomain
		}

		if shouldIgnoreDomain(parts[0], ignoreDomains) {
			return strings.Join([]string{parts[0], parts[1], parts[2]}, "/")
		}

		return strings.Join([]string{prefix, parts[0], parts[1], parts[2]}, "/")
	}
	return name
}

// shouldIgnoreDomain checks if the image domain should be ignored
func shouldIgnoreDomain(domain string, ignoreDomains []string) bool {
	for _, ignoreDomain := range ignoreDomains {
		if domain == ignoreDomain {
			return true
		}
	}
	return false
}

func isDomain(name string) bool {
	return strings.Contains(name, ".")
}

var (
	legacyDefaultDomain = map[string]struct{}{
		"index.docker.io":      {},
		"registry-1.docker.io": {},
	}
)

func isLegacyDefaultDomain(name string) bool {
	_, ok := legacyDefaultDomain[name]
	return ok
}

// ExtractRepository extracts the repository/domain from an image reference
func ExtractRepository(image string) string {
	parts := strings.SplitN(image, "/", 3)
	switch len(parts) {
	case 1:
		// nginx -> docker.io
		return defaultDomain
	case 2:
		// user/repo -> docker.io (if not a domain)
		// domain/repo -> domain
		if !isDomain(parts[0]) {
			return defaultDomain
		}
		if isLegacyDefaultDomain(parts[0]) {
			return defaultDomain
		}
		return parts[0]
	case 3:
		// domain/user/repo -> domain
		// user/repo/tag -> docker.io (if not a domain)
		if !isDomain(parts[0]) {
			return defaultDomain
		}
		if isLegacyDefaultDomain(parts[0]) {
			return defaultDomain
		}
		return parts[0]
	}
	return defaultDomain
}
