package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func MoveIsValid(m string) bool {
	return m == "rock" || m == "paper" || m == "scissors"
}

func CalculateCommitment(move, salt string) string {
	h := sha256.Sum256([]byte(move + ":" + salt))
	return hex.EncodeToString(h[:])
}
