package mobile

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const challengeBytes = 32

type Authenticator struct {
	store ChallengeSecretStore
}

type ChallengeSecretStore interface {
	SecretFor(deviceID string) ([]byte, DeviceCredential, error)
}

func NewAuthenticator(store ChallengeSecretStore) Authenticator {
	return Authenticator{store: store}
}

func NewChallenge() (string, error) {
	buf := make([]byte, challengeBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate auth challenge: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func SignChallenge(secret []byte, challenge string) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("secret is required")
	}
	normalizedChallenge := strings.TrimSpace(challenge)
	if normalizedChallenge == "" {
		return "", errors.New("challenge is required")
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(normalizedChallenge))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (a Authenticator) Verify(deviceID string, challenge string, signature string) (DeviceCredential, error) {
	if a.store == nil {
		return DeviceCredential{}, errors.New("credential store is required")
	}
	secret, device, err := a.store.SecretFor(deviceID)
	if err != nil {
		return DeviceCredential{}, err
	}
	expected, err := SignChallenge(secret, challenge)
	if err != nil {
		return DeviceCredential{}, err
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(strings.TrimSpace(signature))) != 1 {
		return DeviceCredential{}, errors.New("invalid mobile device signature")
	}
	return device, nil
}
