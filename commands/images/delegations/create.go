package delegations

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/lib/images"
	"github.com/daticahealth/cli/lib/prompts"
	notaryUtils "github.com/docker/notary/tuf/utils"
)

const (
	passwordAttempts = 3
	delegationRole   = "user"
)

func cmdDelegationsCreate(envID, keyPath string, size, expiration int, importKey bool, ie environments.IEnvironments, ii images.IImages, ip prompts.IPrompts) error {
	_, err := ie.Retrieve(envID)
	if err != nil {
		return err
	}

	var privateKey *rsa.PrivateKey
	var pemKey *pem.Block
	if keyPath == "" {
		logrus.Println("Generating new key")
		privateKey, err = rsa.GenerateKey(rand.Reader, size)
		if err != nil {
			return err
		}
		var password string
		for i := 0; i < passwordAttempts; i++ {
			password = ip.Password("Enter password for delegation key: ")
			//TODO: Password strength requirements
			confirmation := ip.Password("Repeat password for delegation key: ")
			if password == confirmation {
				break
			}
			logrus.Println("\nEntries do not match. Please try again")
		}

		passwordBytes := []byte(password)
		pemKey, err = x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(privateKey), passwordBytes, x509.PEMCipherAES256)
		if err != nil {
			return err
		}
		pemKey.Headers["role"] = "user"
	} else {
		privateKeyBytes, err := ioutil.ReadFile(keyPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file for delegation key does not exist: %s", keyPath)
			}
			return fmt.Errorf("unable to read delegation key from file: %s", keyPath)
		}
		pemKey, _ = pem.Decode(privateKeyBytes)
		if err != nil {
			return err
		}
		//TODO: Make sure key is encrypted
		keyBytes, err := x509.DecryptPEMBlock(pemKey, []byte(ip.KeyPassphrase("Enter password for the key: ")))
		if err != nil {
			return err
		}
		if role, ok := pemKey.Headers["role"]; !ok || role != delegationRole {
			logrus.Printf("Setting PEM header \"role: %s\"\n", delegationRole)
			pemKey.Headers["role"] = delegationRole
			//TODO: Rewrite the key file
		}
		switch pemKey.Type {
		//TODO: This will probably be ENCRYPTED PRIVATE KEY, need to figure out how to parse type
		case "RSA PRIVATE KEY":
			privateKey, err = x509.ParsePKCS1PrivateKey(keyBytes)
			if err != nil {
				return err
			}
		// case "EC PRIVATE KEY":
		// 	//TODO: Allow any key type
		// 	_, err = x509.ParseECPrivateKey(keyBytes)
		// if err != nil {
		// 	return err
		// }
		default:
			return fmt.Errorf("Invalid key type: %s\n", pemKey.Type)
		}

	}

	logrus.Println("\nThe following questions will gather information to be encoded in your delegation certificate.")
	country := ip.CaptureInput("Country name (2 letter code): ")
	province := ip.CaptureInput("State or Province Name (full name): ")
	locality := ip.CaptureInput("Locality Name (e.g. city): ")
	organization := ip.CaptureInput("Organization Name (e.g. company): ")
	orgUnit := ip.CaptureInput("Organizational Unit Name (e.g. section): ")
	commonName := ip.CaptureInput("Common Name (e.g. user of delegation key): ")
	fmt.Println()

	crtTemplate := &x509.Certificate{
		IsCA: true,
		BasicConstraintsValid: true,
		SubjectKeyId:          []byte{1, 2, 3},
		SerialNumber:          big.NewInt(1234),
		Subject: pkix.Name{
			Country:            []string{country},
			Province:           []string{province},
			Locality:           []string{locality},
			Organization:       []string{organization},
			OrganizationalUnit: []string{orgUnit},
			CommonName:         commonName,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(0, 0, expiration),
	}
	cert, err := x509.CreateCertificate(rand.Reader, crtTemplate, crtTemplate, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	keyID, err := publicKeyID(cert)
	if err != nil {
		return err
	}

	keyFilename := fmt.Sprintf("%s.key", keyID)
	if importKey {
		keyFilename = fmt.Sprintf("%s/%s", images.RootTrustDir(), keyFilename)
	}
	pemfile, _ := os.Create(keyFilename)
	pem.Encode(pemfile, pemKey)
	pemfile.Close()

	certFilename := "delegation.crt"
	if commonName != "" {
		certFilename = fmt.Sprintf("%s_%s", strings.Replace(commonName, " ", "_", -1), certFilename)
	}
	certFile, err := os.Create(certFilename)
	if err != nil {
		return err
	}
	pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	certFile.Close()

	logrus.Printf("Delegation key %s.key and certificate %s created successfully.\n"+
		"Use the certificate to grant signing priveledges to your delgation key for a repository with the `datica images delegations add` command.\n"+
		"Note: The add command reqires access to the root and targets key for the repo.", keyID, certFilename)
	return nil
}

func publicKeyID(cert []byte) (string, error) {
	x509Cert, err := x509.ParseCertificate(cert)
	if err != nil {
		return "", err
	}
	return notaryUtils.CanonicalKeyID(notaryUtils.CertToKey(x509Cert))
}
