package environment

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

type templateEnvironment struct {
	IsProduction          bool
	Namespace             string
	Environment           string
	GithubTeam            string
	SlackChannel          string
	BusinessUnit          string
	Application           string
	InfrastructureSupport string
	TeamName              string
	SourceCode            string
	Owner                 string
	validPath             bool
}

type templateEnvironmentFile struct {
	outputPath string
	content    string
	name       string
	url        string
}

// CreateTemplateNamespace creates the terraform files from environment's template folder
func CreateTemplateNamespace(cmd *cobra.Command, args []string) error {

	templates := []*templateEnvironmentFile{
		{
			url:  "https://raw.githubusercontent.com/ministryofjustice/cloud-platform-environments/cli-template/namespace-resources-cli-template/00-namespace.yaml",
			name: "00-namespace.yaml",
		},
		{
			url:  "https://raw.githubusercontent.com/ministryofjustice/cloud-platform-environments/cli-template/namespace-resources-cli-template/01-rbac.yaml",
			name: "01-rbac.yaml",
		},
		{
			url:  "https://raw.githubusercontent.com/ministryofjustice/cloud-platform-environments/cli-template/namespace-resources-cli-template/02-limitrange.yaml",
			name: "02-limitrange.yaml",
		},
		{
			url:  "https://raw.githubusercontent.com/ministryofjustice/cloud-platform-environments/cli-template/namespace-resources-cli-template/03-resourcequota.yaml",
			name: "03-resourcequota.yaml",
		},
		{
			url:  "https://raw.githubusercontent.com/ministryofjustice/cloud-platform-environments/cli-template/namespace-resources-cli-template/04-networkpolicy.yaml",
			name: "04-networkpolicy.yaml",
		},
		{
			url:  "https://raw.githubusercontent.com/ministryofjustice/cloud-platform-environments/cli-template/namespace-resources-cli-template/resources/main.tf",
			name: "resources/main.tf",
		},
		{
			url:  "https://raw.githubusercontent.com/ministryofjustice/cloud-platform-environments/cli-template/namespace-resources-cli-template/resources/versions.tf",
			name: "resources/versions.tf",
		},
		{
			url:  "https://raw.githubusercontent.com/ministryofjustice/cloud-platform-environments/cli-template/namespace-resources-cli-template/resources/variables.tf",
			name: "resources/variables.tf",
		},
	}

	err := initTemplateNamespace(templates)
	if err != nil {
		return (err)
	}

	namespaceValues, err := templateNamespaceSetValues()
	if err != nil {
		return (err)
	}

	err = setupPaths(templates, namespaceValues.Namespace)
	if err != nil {
		return (err)
	}

	for _, i := range templates {
		t, err := template.New("namespaceTemplates").Parse(i.content)
		if err != nil {
			return err
		}

		f, err := os.Create(i.outputPath)
		if err != nil {
			return err
		}

		err = t.Execute(f, namespaceValues)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Namespace files generated under namespaces/live-1.cloud-platform.service.justice.gov.uk/%s\n", namespaceValues.Namespace)
	color.Info.Tips("Please review before raising PR")

	return nil
}

func templateNamespaceSetValues() (*templateEnvironment, error) {
	values := templateEnvironment{}

	GithubTeams, err := getGitHubTeams()
	if err != nil {
		return nil, err
	}

	Namespace := promptString{
		label:        "What is the name of your namespace? This should be of the form: <application>-<environment>. e.g. myapp-dev (lower-case letters and dashes only)",
		defaultValue: "",
		validation:   "no-spaces",
	}
	err = Namespace.promptString()
	if err != nil {
		return nil, err
	}

	Environment := promptString{
		label:        "What type of application environment is this namespace for? e.g. development, staging, production",
		defaultValue: "",
		validation:   "no-spaces",
	}
	err = Environment.promptString()
	if err != nil {
		return nil, err
	}

	IsProduction := promptYesNo{
		label:        "Is this a production namespace? (please answer true or false)",
		defaultValue: 0,
	}
	err = IsProduction.promptyesNo()
	if err != nil {
		return nil, err
	}

	Application := promptString{
		label:        "What is the name of your application/service? (e.g. Send money to a prisoner)",
		defaultValue: "",
	}
	err = Application.promptString()
	if err != nil {
		return nil, err
	}

	GithubTeam, err := promptSelectGithubTeam(GithubTeams)

	businessUnit := promptString{
		label:        "Which part of the MoJ is responsible for this service? (e.g HMPPS, Legal Aid Agency)",
		defaultValue: "",
	}
	err = businessUnit.promptString()
	if err != nil {
		return nil, err
	}

	SlackChannel := promptString{
		label:        "What is the best slack channel (without the '#') to use if we need to contact your team?\n(If you don't have a team slack channel, please create one)",
		defaultValue: "",
	}
	err = SlackChannel.promptString()
	if err != nil {
		return nil, err
	}

	teamName := promptString{label: "Team's name", defaultValue: "", validation: "no-spaces"}
	err = teamName.promptString()
	if err != nil {
		return nil, err
	}

	InfrastructureSupport := promptString{
		label:        "What is the email address for the team which owns the application? (this should not be a named individual's email address)",
		defaultValue: "",
		validation:   "email",
	}
	err = InfrastructureSupport.promptString()
	if err != nil {
		return nil, err
	}

	SourceCode := promptString{
		label:        "What is the Github repository URL of the source code for this application?",
		defaultValue: "",
		validation:   "url",
	}
	err = SourceCode.promptString()
	if err != nil {
		return nil, err
	}

	Owner := promptString{
		label:        "Which team in your organisation is responsible for this application? (e.g. Sentence Planning)",
		defaultValue: "",
	}
	err = Owner.promptString()
	if err != nil {
		return nil, err
	}

	values.Application = Application.value
	values.BusinessUnit = businessUnit.value
	values.Namespace = Namespace.value
	values.GithubTeam = GithubTeam
	values.Environment = Environment.value
	values.IsProduction = IsProduction.value
	values.SlackChannel = SlackChannel.value
	values.InfrastructureSupport = InfrastructureSupport.value
	values.SourceCode = SourceCode.value
	values.Owner = Owner.value
	values.TeamName = teamName.value

	return &values, nil
}

func initTemplateNamespace(t []*templateEnvironmentFile) error {
	for _, s := range t {
		content, err := downloadTemplate(s.url)
		if err != nil {
			return err
		}
		s.content = content
	}

	return nil
}

func setupPaths(t []*templateEnvironmentFile, namespace string) error {
	path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return errors.New("You are outside cloud-platform-environment repo")
	}
	fullPath := strings.TrimSpace(string(path))
	for _, s := range t {
		s.outputPath = fullPath + fmt.Sprintf("/namespaces/live-1.cloud-platform.service.justice.gov.uk/%s/", namespace) + s.name
	}

	err = os.Mkdir(fullPath+fmt.Sprintf("/namespaces/live-1.cloud-platform.service.justice.gov.uk/%s/", namespace), 0755)
	if err != nil {
		return err
	}
	err = os.Mkdir(fullPath+fmt.Sprintf("/namespaces/live-1.cloud-platform.service.justice.gov.uk/%s/resources", namespace), 0755)
	if err != nil {
		return err
	}

	return nil
}
