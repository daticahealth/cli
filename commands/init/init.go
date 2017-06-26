package init

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"golang.org/x/crypto/ssh"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/deploykeys"
	"github.com/daticahealth/cli/commands/domain"
	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/commands/git"
	"github.com/daticahealth/cli/commands/keys"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/commands/sites"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	homedir "github.com/mitchellh/go-homedir"
)

func CmdInit(settings *models.Settings, p prompts.IPrompts) error {
	logrus.Println("To set up your local repository, we need to know what environment and service you want to push your code to.")

	ie := environments.New(settings)
	envs, errs := ie.List()
	if errs != nil && len(errs) > 0 {
		logrus.Debugf("Error listing environments: %+v", errs)
	}
	if envs == nil || len(*envs) == 0 {
		logrus.Println("You don't have any environments. Visit https://datica.com/compliant-cloud to get started")
		return nil
	}

	config.StoreEnvironments(envs, settings)

	logrus.Printf("%d environment(s) found:", len(*envs))
	for i, env := range *envs {
		logrus.Printf("\t%d) %s", i+1, env.Name)
	}
	env := (*envs)[0]
	if len(*envs) > 1 {
		for {
			choice := p.CaptureInput("Enter your choice as a number: ")
			i, err := strconv.ParseUint(choice, 10, 64)
			if err != nil || i == 0 || i > uint64(len(*envs)) {
				logrus.Printf("%s is not a valid number", choice)
				continue
			}
			env = (*envs)[i-1]
			break
		}
	}
	settings.EnvironmentID = env.ID
	settings.Pod = env.Pod
	settings.EnvironmentName = env.Name
	settings.OrgID = env.OrgID
	logrus.Printf("Initializing %s...", env.Name)

	is := services.New(settings)
	svcs, err := is.ListByEnvID(env.ID, env.Pod)
	if err != nil {
		return err
	}
	codeServices := []models.Service{}
	for _, svc := range *svcs {
		if svc.Type == "code" {
			codeServices = append(codeServices, svc)
		}
	}
	if len(codeServices) == 0 {
		logrus.Println("You don't have any code services. Visit the dashboard to add one")
		return nil
	}
	logrus.Printf("%d code service(s) found for %s:", len(codeServices), env.Name)
	for i, svc := range codeServices {
		logrus.Printf("\t%d) %s", i+1, svc.Label)
	}
	svc := codeServices[0]
	if len(codeServices) > 1 {
		for {
			choice := p.CaptureInput("Enter your choice as a number: ")
			i, err := strconv.ParseUint(choice, 10, 64)
			if err != nil || i == 0 || i > uint64(len(*envs)) {
				logrus.Printf("%s is not a valid number", choice)
				continue
			}
			svc = codeServices[i-1]
			break
		}
	}

	ig := git.New()
	if !ig.Exists() {
		logrus.Println("Initializing a local git repo...")
		ig.Create()
	}

	logrus.Printf("Adding git remote for %s...", svc.Label)
	remoteName := "datica"
	remotes, err := ig.List()
	if err != nil {
		return err
	}
	exists := false
	for _, r := range remotes {
		if r == remoteName {
			exists = true
			break
		}
	}
	if exists {
		if err := p.YesNo(fmt.Sprintf("A git remote named \"%s\" already exists.", remoteName), "Would you like to overwrite it? (y/n) "); err != nil {
			return err
		}
		err = ig.SetURL(remoteName, svc.Source)
	} else {
		err = ig.Add(remoteName, svc.Source)
	}
	if err != nil {
		return fmt.Errorf("Failed to setup the git remote: %s", err)
	}

	// TODO insert lets encrypt setup here once ready
	logrus.Println("Creating certificates...")

	ik := keys.New(settings)
	userKeys, err := ik.List()
	if err != nil {
		return err
	}
	if userKeys == nil || len(*userKeys) == 0 {
		logrus.Println("You'll need to add an SSH key in order to push code.")
		for {
			keyPath := p.CaptureInput("Enter the path to your public SSH key (leave empty to skip): ")
			if keyPath == "" {
				break
			} else if _, err = os.Stat(keyPath); os.IsNotExist(err) {
				keyPath, _ = homedir.Expand(keyPath)
				if _, err = os.Stat(keyPath); os.IsNotExist(err) {
					logrus.Printf("A file does not exist at %s", keyPath)
					continue
				}
			}

			keyBytes, err := ioutil.ReadFile(keyPath)
			if err != nil {
				logrus.Printf("Could not read file at %s", keyPath)
				continue
			}
			k, err := deploykeys.New(settings).ParsePublicKey(keyBytes)
			if err != nil {
				logrus.Printf("A valid public SSH key does not exist at %s", keyPath)
				continue
			}
			err = ik.Add("my-key", string(ssh.MarshalAuthorizedKey(k)))
			if err != nil {
				return err
			}
			logrus.Println("Successfully added your SSH key.")
			break
		}
	}

	domain, err := domain.FindEnvironmentDomain(env.ID, env.Namespace, is, sites.New(settings))
	if err != nil {
		return err
	}
	logrus.Printf("All set! Run \"git push datica master\" to push your code. Once the build finishes, you can view your code running at %s. It may take a few minutes for DNS to propagate.", domain)
	return nil
}
