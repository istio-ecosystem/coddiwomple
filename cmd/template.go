package cmd

import (
	"fmt"
	"os"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func templateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "template",
		Short: "mcc template creates manifests",
		Long: `mcc template creates templates for Istio
	https://github.com/tetratelabs/mcc   `,
		Run: func(cmd *cobra.Command, args []string) {
			generate(args...)
		},
	}

}

type Parameters map[string]interface{}

// move to external configuration
func getParameters() Parameters {
	params := Parameters{
		"resolution": "STATIC",
	}

	params["gateway"] = Parameters{
		"name": "bookinfo",
	}

	params["cluster"] = Parameters{
		"instanceID": "spcedl94wx",
		"tunnelPort": 80,
	}

	params["remote"] = Parameters{
		"cluster": Parameters{
			"instanceID": "spcwr1q2cu",
		},
	}

	params["service"] = Parameters{
		"name":      "reviews",
		"namespace": "default",
	}

	d, err := yaml.Marshal(&params)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Debugf("--- Parameter dump:\n%s\n\n", string(d))

	return params
}

func generate(args ...string) {
	var templates = template.Must(template.ParseGlob("templates/*"))
	params := getParameters()
	for _, t := range templates.Templates() {
		fmt.Println("---")

		err := templates.ExecuteTemplate(os.Stdout, t.Name(), params)
		if err != nil {
			log.Fatal("Cannot generate ", err)
		}
	}

}
