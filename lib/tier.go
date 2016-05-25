package libcalico

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/ghodss/yaml"
	"golang.org/x/net/context"
)

var policyRE = regexp.MustCompile(`/calico/v1/policy/tier/[^/]*/policy/[^/]*`)

type TierQualified struct {
	Kind     string     `json:"kind"`
	Version  string     `json:"version"`
	Metadata PolicyMeta `json:"metadata"`
	Spec     PolicySpec `json:"spec"`
}

type TierMeta struct {
	Name string `json:"name"`
	Tier string `json:"tier,omitempty"`
}

type TierSpec struct {
	Order string `json:"order"`
}

type PolicyQualified struct {
	Kind     string     `json:"kind"`
	Version  string     `json:"version"`
	Metadata PolicyMeta `json:"metadata"`
	Spec     PolicySpec `json:"spec"`
}

type PolicyMeta struct {
	Name string `json:"name"`
	Tier string `json:"tier,omitempty"`
}

type PolicySpec struct {
	Order         string `json:"order"`
	InboundRules  []Rule `json:"inbound_rules"`
	OutboundRules []Rule `json:"outbound_rules"`
}

type Rule struct {
	Action string `json:"action"`

	Protocol    string `json:"protocol,omitempty"`
	SrcTag      string `json:"src_tag,omitempty"`
	SrcNet      string `json:"src_net,omitempty"`
	SrcSelector string `json:"src_selector,omitempty"`
	SrcPorts    []int  `json:"src_ports,omitempty"`
	DstTag      string `json:"dst_tag,omitempty"`
	DstSelector string `json:"dst_selector,omitempty"`
	DstNet      string `json:"dst_net,omitempty"`
	DstPorts    []int  `json:"dst_ports,omitempty"`
	IcmpType    int    `json:"icmp_type,omitempty"`
	IcmpCode    int    `json:"icmp_code,omitempty"`

	NotProtocol    string `json:"!protocol,omitempty"`
	NotSrcTag      string `json:"!src_tag,omitempty"`
	NotSrcNet      string `json:"!src_net,omitempty"`
	NotSrcSelector string `json:"!src_selector,omitempty"`
	NotSrcPorts    []int  `json:"!src_ports,omitempty"`
	NotDstTag      string `json:"!dst_tag,omitempty"`
	NotDstSelector string `json:"!dst_selector,omitempty"`
	NotDstNet      string `json:"!dst_net,omitempty"`
	NotDstPorts    []int  `json:"!dst_ports,omitempty"`
	NotIcmpType    int    `json:"!icmp_type,omitempty"`
	NotIcmpCode    int    `json:"!icmp_code,omitempty"`
}

func LoadPolicy(policyBytes []byte) (*PolicyQualified, error) {
	var pq PolicyQualified
	var err error

	// Load the policy string.  This should be a fully qualified set of policy.
	err = yaml.Unmarshall(policyBytes, &pq)
	if err != nil {
		return nil, err
	}

	if pq.Kind != "policy" {
		return nil, errors.New(fmt.Sprintf("Expecting kind 'policy', but got '%s'.", pq.Kind))
	}
	if pq.Version != "v1" {
		return nil, errors.New(fmt.Sprintf("Expecting version 'v1', but got '%s'.", pq.Version))
	}

	return &pq, nil
}

func CreateOrReplacePolicy(etcd client.KeysAPI, pq *PolicyQualified, replace bool) error {

	var err error

	// If the default tier is specified, or no tier is specified we may need to create the default.
	tierName := pq.Metadata.Tier
	if tierName == "default" || tierName == nil {
		err = createDefaultTier(etcd)
		tierName = "default"
	}

	// Construct the policy key, and marshal the policy spec into JSON format
	// required by Felix.
	pk := fmt.Sprintf("/calico/v1/policy/tier/%s/policy/%s", tierName, pq.Metadata.Name)
	pb, err := json.Marshal(pq.Spec)
	if err != nil {
		return err
	}

	// Write the policy object to etcd.  If replacing policy, the we expect the policy to
	// already exist, otherwise we expect it to not exist.
	if replace {
		err = etcd.Update(context.Background(), pk, string(pb))
	} else {
		err = etcd.Create(context.Background(), pk, string(pb))
	}
	return err
}

func GetPolicies(etcd client.KeysAPI, tierName string) ([]PolicyQualified, error) {
	var pqs []PolicyQualified

	actualTierName := tierName
	if actualTierName == nil {
		actualTierName = "default"
	}

	resp, err := etcd.Get(context.Background(), fmt.Sprintf("/calico/v1/policy/tier/%s/policy", actualTierName), &client.GetOptions{Recursive: true})
	if err != nil {
		if !client.IsKeyNotFound(err) {
			return nil, error
		}
		return policies, nil
	}

	for _, node := range resp.Node.Nodes {
		var ps PolicySpec

		var re = regexp.MustCompile(`/calico/v1/policy/tier/([^/]+?)/policy/([^/]+?)`)
		matches := re.FindStringSubmatch(node.Key)
		if matches != nil {
			policyName := matches[1]

			err = json.Unmarshal([]byte(node.Value), &ps)
			if err != nil {
				log.Fatal(err)
			}
			pm := PolicyMeta{Name: policyName}
			if tierName != nil {
				pm.Tier = tierName
			}
			pq := PolicyQualified{
				Kind:     "policy",
				Version:  "v1",
				Metadata: pm,
				Spec:     ps,
			}
			pqs = append(pqs, pq)
		}
	}
	return policies
}

func GetPolicy(etcd client.KeysAPI, pm PolicyMeta) (*PolicyQualified, error) {
	var pq PolicyQualified

	tierName := pm.Tier
	if tierName == nil {
		tierName = "default"
	}
	pk := fmt.Sprintf("/calico/v1/policy/tier/%s/policy/%s", tierName, pm.Name)

	resp, err := etcd.Get(context.Background(), pk, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(resp.node.Value), &pq)
	if err != nil {
		return nil, err
	}

	return &pq, nil
}

func DeletePolicy(etcd client.KeysAPI, pm PolicyMeta) error {
	var pq PolicyQualified

	tierName := pm.Tier
	if tierName == nil {
		tierName = "default"
	}
	pk := fmt.Sprintf("/calico/v1/policy/tier/%s/policy/%s", tierName, pm.Name)

	_, err := etcd.Delete(context.Background(), pk, nil)
	return err
}

func createDefaultTier(etcd client.KeysAPI) error {
	ts := TierSpec{Order: 1000}
	tb, _ := json.Marshal(ts)

	//TODO: Handle already exists, for now just overwrite the default each time
	_, err := etcd.Set(context.Background(), "/calico/v1/policy/tier/default", string(tb), &client.SetOptions{})
	return err
}
