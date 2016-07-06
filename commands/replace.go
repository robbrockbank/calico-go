package commands

import (
	"github.com/docopt/docopt-go"

	"fmt"

	"github.com/golang/glog"
	"github.com/projectcalico/libcalico/lib/api"
	"github.com/projectcalico/libcalico/lib/api/unversioned"
	"github.com/projectcalico/libcalico/lib/client"
)

func Replace(args []string) error {
	doc := EtcdIntro + `Replace a resource by filename or stdin.

If replacing an existing resource, the complete resource spec must be provided. This can be obtained by
$ calicoctl get -o yaml <TYPE> <NAME>

Usage:
  calicoctl replace --filename=<FILENAME> [--config=<CONFIG>]

Examples:
  # Replace a policy using the data in policy.yaml.
  calicoctl replace -f ./policy.yaml

  # Replace a pod based on the YAML passed into stdin.
  cat policy.yaml | calicoctl replace -f -

Options:
  -f --filename=<FILENAME>     Filename to use to replace the resource.  If set to "-" loads from stdin.
  -c --config=<CONFIG>         Filename containing connection configuration in YAML or JSON format.
                               [default: /etc/calico/calicoctl.cfg]
`
	parsedArgs, err := docopt.Parse(doc, args, true, "calicoctl", false, false)
	if err != nil {
		return err
	}
	if len(parsedArgs) == 0 {
		return nil
	}

	cmd := replace{}
	results := executeConfigCommand(parsedArgs, cmd)
	glog.V(2).Infof("results: %v", results)

	if results.fileInvalid {
		fmt.Printf("Error processing input file: %v\n", results.err)
	} else if results.numHandled == 0 {
		if results.numResources == 0 {
			fmt.Printf("No resources specified in file\n")
		} else if results.numResources == 1 {
			fmt.Printf("Failed to replace '%s' resource: %v\n", results.singleKind, results.err)
		} else if results.singleKind != "" {
			fmt.Printf("Failed to any replace '%s' resources: %v\n", results.singleKind, results.err)
		} else {
			fmt.Printf("Failed to any replace 'resources: %v\n", results.err)
		}
	} else if results.err == nil {
		if results.singleKind != "" {
			fmt.Printf("Successfully replaced %d '%s' resource(s)\n", results.numHandled, results.singleKind)
		} else {
			fmt.Printf("Successfully replaced %d resource(s)\n", results.numHandled)
		}
	} else {
		fmt.Printf("Partial success: ")
		if results.singleKind != "" {
			fmt.Printf("replaced the first %d out of %d '%s' resources:\n",
				results.numHandled, results.numResources, results.singleKind)
		} else {
			fmt.Printf("replaced the first %d out of %d resources:\n",
				results.numHandled, results.numResources)
		}
		fmt.Printf("Hit error: %v\n", results.err)
	}

	return results.err
}

// commandInterface for replace command.
// Maps the generic resource types to the typed client interface.
type replace struct {
}

func (c replace) execute(client *client.Client, resource unversioned.Resource) (unversioned.Resource, error) {
	var err error
	switch r := resource.(type) {
	case api.HostEndpoint:
		_, err = client.HostEndpoints().Update(&r)
	case api.Policy:
		_, err = client.Policies().Update(&r)
	case api.Profile:
		_, err = client.Profiles().Update(&r)
	case api.Tier:
		_, err = client.Tiers().Update(&r)
	default:
		panic(fmt.Errorf("Unhandled resource type: %v", resource))
	}

	return resource, err
}
