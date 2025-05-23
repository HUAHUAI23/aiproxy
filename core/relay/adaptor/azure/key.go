package azure

import (
	"errors"
	"strings"

	"github.com/labring/aiproxy/core/relay/adaptor"
)

var _ adaptor.KeyValidator = (*Adaptor)(nil)

func (a *Adaptor) ValidateKey(key string) error {
	_, _, err := getTokenAndAPIVersion(key)
	if err != nil {
		return err
	}
	return nil
}

func (a *Adaptor) KeyHelp() string {
	return "key or key|api-version"
}

const defaultAPIVersion = "2024-12-01-preview"

func getTokenAndAPIVersion(key string) (string, string, error) {
	split := strings.Split(key, "|")
	if len(split) == 1 {
		return key, defaultAPIVersion, nil
	}
	if len(split) != 2 {
		return "", "", errors.New("invalid key format")
	}
	return split[0], split[1], nil
}
