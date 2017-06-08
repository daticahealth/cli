package prompts

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/daticahealth/cli/config"

	"golang.org/x/crypto/ssh/terminal"
)

// IPrompts is the interface in which to interact with the user and accept
// input.
type IPrompts interface {
	EmailPassword(existingEmail, existingPassword string) (string, string, error)
	KeyPassphrase(string) string
	Password(msg string) string
	PHI() error
	YesNo(msg, prompt string) error
	OTP(string) string
	GenericPrompt(msg, prompt string, validOptions []string) string
	CaptureInput(msg string) string
}

// SPrompts is a concrete implementation of IPrompts
type SPrompts struct{}

// New returns a new instance of IPrompts
func New() IPrompts {
	return &SPrompts{}
}

// EmailPassword prompts a user to enter their email and password.
func (p *SPrompts) EmailPassword(existingEmail, existingPassword string) (string, string, error) {
	email := existingEmail
	var err error
	if email == "" {
		fmt.Print("Email: ")
		in := bufio.NewReader(os.Stdin)
		email, err = in.ReadString('\n')
		if err != nil {
			return "", "", errors.New("Invalid email")
		}
		email = strings.TrimRight(email, "\n")
		if runtime.GOOS == "windows" {
			email = strings.TrimRight(email, "\r")
		}
	} else {
		fmt.Printf("Using email from environment variable %s\n", config.DaticaEmailEnvVar)
	}
	password := existingPassword
	if password == "" {
		fmt.Print("Password: ")
		bytes, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println("")
		password = string(bytes)
	} else {
		fmt.Printf("Using password from environment variable %s\n", config.DaticaPasswordEnvVar)
	}
	return email, password, nil
}

// KeyPassphrase prompts a user to enter a passphrase for a named key.
func (p *SPrompts) KeyPassphrase(filepath string) string {
	fmt.Printf("Enter passphrase for %s: ", filepath)
	bytes, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println("")
	return string(bytes)
}

// PHI prompts a user to accept liability for downloading PHI to their local
// machine.
func (p *SPrompts) PHI() error {
	acceptAnswers := []string{"y", "yes"}
	denyAnswers := []string{"n", "no"}

	answer := p.GenericPrompt("This operation might result in PHI data being downloaded and decrypted to your local machine. By entering \"y\" at the prompt below, you warrant that you have the necessary privileges to view the data, have taken all necessary precautions to secure this data, and absolve Datica of any issues that might arise from its loss.", "Do you wish to proceed? (y/n) ", append(acceptAnswers, denyAnswers...))
	for _, denyAnswer := range denyAnswers {
		if denyAnswer == strings.ToLower(answer) {
			return fmt.Errorf("Exiting")
		}
	}
	return nil
}

// YesNo outputs a given message and waits for a user to answer `y/n`.
// If yes, flow continues as normal. If no, an error is returned. The given
// message SHOULD contain the string "(y/n)" or some other form of y/n
// indicating that the user needs to type in y or n. This method does not do
// that for you. The message will not have a new line appended to it. If you
// require a newline, add this to the given message.
func (p *SPrompts) YesNo(msg, prompt string) error {
	acceptAnswers := []string{"y", "yes"}
	denyAnswers := []string{"n", "no"}

	answer := p.GenericPrompt(msg, prompt, append(acceptAnswers, denyAnswers...))
	for _, denyAnswer := range denyAnswers {
		if denyAnswer == strings.ToLower(answer) {
			return fmt.Errorf("Exiting")
		}
	}
	return nil
}

// Password prompts the user for a password displaying the given message.
// The password will be hidden while typed. A newline is not added to the given
// message. If a newline is required, it should be part of the passed in string.
func (p *SPrompts) Password(msg string) string {
	fmt.Print(msg)
	bytes, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println("")
	return string(bytes)
}

// OTP prompts for a one-time password and returns the value.
func (p *SPrompts) OTP(preferredMode string) string {
	fmt.Println("This account has two-factor authentication enabled.")
	prompt := "Your one-time password: "
	if preferredMode == "authenticator" {
		prompt = "Your authenticator one-time password: "
	} else if preferredMode == "email" {
		prompt = "One-time password (sent to your email): "
	}
	fmt.Print(prompt)
	var token string
	fmt.Scanln(&token)
	return strings.TrimSpace(token)
}

// GenericPrompt prompts the user and validates the input against the list of
// given case-insensitive valid options. The user's choice is returned.
func (p *SPrompts) GenericPrompt(msg, prompt string, validOptions []string) string {
	var answer string
	fmt.Println(msg)
	for {
		fmt.Printf(prompt)
		fmt.Scanln(&answer)
		fmt.Println("")
		valid := false
		for _, choice := range validOptions {
			if strings.ToLower(choice) == strings.ToLower(answer) {
				valid = true
				break
			}
		}
		if !valid {
			fmt.Printf("%s is not a valid option. Please enter one of %s\n", answer, strings.Join(validOptions, ", "))
		} else {
			break
		}
	}
	return answer
}

// CaptureInput prompts the user with the given msg and reads input until a newline is encountered. The input is
// returned with newlines stripped. The prompt and the input will be on the same line when shown to the user.
func (p *SPrompts) CaptureInput(msg string) string {
	var answer string
	fmt.Printf(msg)
	fmt.Scanln(&answer)
	fmt.Println("")
	return answer
}
