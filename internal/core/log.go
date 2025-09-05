package core

import (
	"fmt"
	"strings"
)

var Log_enable = true

func ErrorLog(s ...string) {
	if Log_enable {
		fmt.Printf(`
****************************************************
error occured!!
%s
****************************************************
`, strings.Join(s, " "))
	}
}
func Log(s ...string) {
	if Log_enable {
		fmt.Print(strings.Join(s, " "))
	}
}
