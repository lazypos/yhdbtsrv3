package yhdbt

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	pay_type_one  = "1"
	pay_type_ten  = "2"
	pay_type_harf = "3"
	pay_type_full = "4"
)

func CheckUser(user string) error {
	if !regexp.MustCompile(`^[0-9a-zA-Z]{6,16}$`).MatchString(user) {
		return fmt.Errorf(`user format error.`)
	}
	return nil
}

func CheckPass(pass string) error {
	passup := strings.ToUpper(pass)
	if !regexp.MustCompile(`^[0-9A-Z]{32}$`).MatchString(passup) {
		return fmt.Errorf(`pass format error.`)
	}
	return nil
}

func CheckNick(nickname string) error {
	if len(nickname) < 6 || len(nickname) > 14 {
		return fmt.Errorf(`nickname length error.`)
	}
	if !regexp.MustCompile(`[\.|\\|/|\?|\(|\)|\[|\]]+`).MatchString(nickname) {
		return fmt.Errorf(`nickname format error.`)
	}
	return nil
}

func CheckPay(pay string) int {
	counts := 0
	switch pay {
	case "1":
		counts = 100
	case "2":
		counts = 1000
	case "3":
		counts = 5000
	case "4":
		counts = 10000
	default:
		counts = 0
	}
	return counts
}
