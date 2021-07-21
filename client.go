package main

import (
	"crypto/ed25519"
	"net"

	"encoding/hex"
	"encoding/json"

	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/zaibon/httpsig"
)

var (
	// ErrRequestFailure is returned if client fails to send
	// the request mostly duo to network problem. Mostly
	// this error is resolved by retrying again later.
	ErrRequestFailure = fmt.Errorf("request failure")

	successCodes = []int{
		http.StatusOK,
		http.StatusCreated,
	}
)

type PriceCurrencyEnum uint8

const (
	PriceCurrencyEUR PriceCurrencyEnum = iota
	PriceCurrencyUSD
	PriceCurrencyTFT
	PriceCurrencyAED
	PriceCurrencyGBP
)

func (e PriceCurrencyEnum) String() string {
	switch e {
	case PriceCurrencyEUR:
		return "EUR"
	case PriceCurrencyUSD:
		return "USD"
	case PriceCurrencyTFT:
		return "TFT"
	case PriceCurrencyAED:
		return "AED"
	case PriceCurrencyGBP:
		return "GBP"
	}
	return "UNKNOWN"
}

type IfaceTypeEnum uint8

const (
	IfaceTypeMacvlan IfaceTypeEnum = iota
	IfaceTypeVlan
)

func (e IfaceTypeEnum) String() string {
	switch e {
	case IfaceTypeMacvlan:
		return "macvlan"
	case IfaceTypeVlan:
		return "vlan"
	}
	return "UNKNOWN"
}

type WalletAddress struct {
	Asset   string `bson:"asset" json:"asset"`
	Address string `bson:"address" json:"address"`
}
type Location struct {
	City      string  `bson:"city" json:"city"`
	Country   string  `bson:"country" json:"country"`
	Continent string  `bson:"continent" json:"continent"`
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
}
type NodeResourcePrice struct {
	Currency PriceCurrencyEnum `bson:"currency" json:"currency"`
	Cru      float64           `bson:"cru" json:"cru"`
	Mru      float64           `bson:"mru" json:"mru"`
	Hru      float64           `bson:"hru" json:"hru"`
	Sru      float64           `bson:"sru" json:"sru"`
	Nru      float64           `bson:"nru" json:"nru"`
}

// PublicIP structure
type PublicIP struct {
	Address       string `bson:"address" json:"address"`
	Gateway       string `bson:"gateway" json:"gateway"`
	ReservationID int64  `bson:"reservation_id" json:"reservation_id"`
}
type NodeCloudUnitPrice struct {
	Currency PriceCurrencyEnum `bson:"currency" json:"currency"`
	CU       float64           `bson:"cu" json:"cu"`
	SU       float64           `bson:"su" json:"su"`
	NU       float64           `bson:"nu" json:"nu"`
	IPv4U    float64           `bson:"ipv4u" json:"ipv4u"`
}

type Farm struct {
	ID                  int64               `bson:"_id" json:"id"`
	ThreebotID          int64               `bson:"threebot_id" json:"threebot_id"`
	IyoOrganization     string              `bson:"iyo_organization" json:"iyo_organization"`
	Name                string              `bson:"name" json:"name"`
	WalletAddresses     []WalletAddress     `bson:"wallet_addresses" json:"wallet_addresses"`
	Location            Location            `bson:"location" json:"location"`
	Email               string              `bson:"email" json:"email"`
	ResourcePrices      []NodeResourcePrice `bson:"resource_prices" json:"resource_prices"`
	PrefixZero          string              `bson:"prefix_zero" json:"prefix_zero"`
	IPAddresses         []PublicIP          `bson:"ipaddresses" json:"ipaddresses"`
	EnableCustomPricing bool                `bson:"enable_custom_pricing" json:"enable_custom_pricing"`
	FarmCloudUnitsPrice NodeCloudUnitPrice  `bson:"farm_cloudunits_price" json:"farm_cloudunits_price"`

	// Grid3 pricing enabled
	IsGrid3Compliant bool `bson:"is_grid3_compliant" json:"is_grid3_compliant"`
}

type User struct {
	ID              int64           `bson:"_id" json:"id"`
	Name            string          `bson:"name" json:"name"`
	Email           string          `bson:"email" json:"email"`
	Pubkey          string          `bson:"pubkey" json:"pubkey"`
	Host            string          `bson:"host" json:"host"`
	Description     string          `bson:"description" json:"description"`
	WalletAddresses []WalletAddress `bson:"wallet_addresses" json:"wallet_addresses"`
	Signature       string          `bson:"-" json:"signature,omitempty"`

	// Trusted Sales channel
	// is a special flag that is only set by TF, if set this user can
	// - sponsors pools
	// - get special discount
	IsTrustedChannel bool `bson:"trusted_sales_channel" json:"trusted_sales_channel"`
}

type ResourceAmount struct {
	Cru uint64  `bson:"cru" json:"cru"`
	Mru float64 `bson:"mru" json:"mru"`
	Hru float64 `bson:"hru" json:"hru"`
	Sru float64 `bson:"sru" json:"sru"`
}

type WorkloadAmount struct {
	Network         uint16 `bson:"network" json:"network"`
	NetworkResource uint16 `bson:"network_resource" json:"network_resource"`
	Volume          uint16 `bson:"volume" json:"volume"`
	ZDBNamespace    uint16 `bson:"zdb_namespace" json:"zdb_namespace"`
	Container       uint16 `bson:"container" json:"container"`
	K8sVM           uint16 `bson:"k8s_vm" json:"k8s_vm"`
	GenericVM       uint16 `bson:"generic_vm" json:"generic_vm"`
	Proxy           uint16 `bson:"proxy" json:"proxy"`
	ReverseProxy    uint16 `bson:"reverse_proxy" json:"reverse_proxy"`
	Subdomain       uint16 `bson:"subdomain" json:"subdomain"`
	DelegateDomain  uint16 `bson:"delegate_domain" json:"delegate_domain"`
}

type Proof struct {
	Created      string                 `bson:"created" json:"created"`
	HardwareHash string                 `bson:"hardware_hash" json:"hardware_hash"`
	DiskHash     string                 `bson:"disk_hash" json:"disk_hash"`
	Hardware     map[string]interface{} `bson:"hardware" json:"hardware"`
	Disks        map[string]interface{} `bson:"disks" json:"disks"`
	Hypervisor   []string               `bson:"hypervisor" json:"hypervisor"`
}

type Node struct {
	ID                int64          `bson:"_id" json:"id"`
	NodeId            string         `bson:"node_id" json:"node_id"`
	HostName          string         `bson:"hostname" json:"hostname"`
	NodeIdV1          string         `bson:"node_id_v1" json:"node_id_v1"`
	FarmId            int64          `bson:"farm_id" json:"farm_id"`
	OsVersion         string         `bson:"os_version" json:"os_version"`
	Created           string         `bson:"created" json:"created"`
	Updated           string         `bson:"updated" json:"updated"`
	Uptime            int64          `bson:"uptime" json:"uptime"`
	Address           string         `bson:"address" json:"address"`
	Location          Location       `bson:"location" json:"location"`
	TotalResources    ResourceAmount `bson:"total_resources" json:"total_resources"`
	UsedResources     ResourceAmount `bson:"used_resources" json:"used_resources"`
	ReservedResources ResourceAmount `bson:"reserved_resources" json:"reserved_resources"`
	Workloads         WorkloadAmount `bson:"workloads" json:"workloads"`
	Proofs            []Proof        `bson:"proofs" json:"proofs"`
	Ifaces            []Iface        `bson:"ifaces" json:"ifaces"`
	PublicConfig      *PublicIface   `bson:"public_config,omitempty" json:"public_config"`
	FreeToUse         bool           `bson:"free_to_use" json:"free_to_use"`
	Approved          bool           `bson:"approved" json:"approved"`
	PublicKeyHex      string         `bson:"public_key_hex" json:"public_key_hex"`
	WgPorts           []int64        `bson:"wg_ports" json:"wg_ports"`
	Deleted           bool           `bson:"deleted" json:"deleted"`
	Reserved          bool           `bson:"reserved" json:"reserved"`
}

type Iface struct {
	Name       string   `bson:"name" json:"name"`
	Addrs      []string `bson:"addrs" json:"addrs"`
	Gateway    []net.IP `bson:"gateway" json:"gateway"`
	MacAddress string   `bson:"macaddress" json:"macaddress"`
}

type PublicIface struct {
	Master  string        `bson:"master" json:"master"`
	Type    IfaceTypeEnum `bson:"type" json:"type"`
	Ipv4    string        `bson:"ipv4" json:"ipv4"`
	Ipv6    string        `bson:"ipv6" json:"ipv6"`
	Gw4     net.IP        `bson:"gw4" json:"gw4"`
	Gw6     net.IP        `bson:"gw6" json:"gw6"`
	Version int64         `bson:"version" json:"version"`
}

func NewUser() (User, error) {
	const value = "{}"
	var object User
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type httpClient struct {
	u        *url.URL
	cl       http.Client
	signer   *httpsig.Signer
	identity string
}

// HTTPError is the error type returned by the client
// it contains the error and the HTTP response
type HTTPError struct {
	resp *http.Response
	err  error
}

func (h HTTPError) Error() string {
	return fmt.Sprintf("%v status:%s", h.err, h.resp.Status)
}

// Response return the HTTP response that trigger this error
func (h HTTPError) Response() http.Response {
	return *h.resp
}

func newHTTPClient(raw string, id Identity) (*httpClient, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if !strings.HasSuffix(u.Path, "/api/v1") {
		u.Path = "/api/v1"
	}

	client := &httpClient{
		u: u,
	}

	if id != nil {
		client.signer = httpsig.NewSigner(id.Identity(), id.PrivateKey(), httpsig.Ed25519, []string{"(created)", "date", "threebot-id"})
		client.identity = id.Identity()
	}

	return client, nil
}

func (c *httpClient) url(p ...string) string {
	b := *c.u
	allParts := []string{b.Path}
	allParts = append(allParts, p...)
	b.Path = strings.Join(allParts, "/")
	fmt.Println(b)
	return b.String()

}

func (c *httpClient) sign(r *http.Request) error {
	if c.signer == nil {
		return nil
	}

	r.Header.Set(http.CanonicalHeaderKey("threebot-id"), c.identity)
	return c.signer.Sign(r)
}

func (c *httpClient) process(response *http.Response, output interface{}, expect ...int) error {
	defer response.Body.Close()

	if len(expect) == 0 {
		expect = successCodes
	}

	in := func(i int, l []int) bool {
		for _, x := range l {
			if x == i {
				return true
			}
		}
		return false
	}

	dec := json.NewDecoder(response.Body)
	if !in(response.StatusCode, expect) {
		var output struct {
			E string `json:"error"`
		}

		if err := dec.Decode(&output); err != nil {
			return errors.Wrapf(HTTPError{
				err:  err,
				resp: response,
			}, "failed to load error while processing invalid return code of: %s", response.Status)
		}

		return HTTPError{
			err:  fmt.Errorf(output.E),
			resp: response,
		}
	}

	if output == nil {
		//discard output
		ioutil.ReadAll(response.Body)
		return nil
	}

	if err := dec.Decode(output); err != nil {
		return HTTPError{
			err:  errors.Wrap(err, "failed to load output"),
			resp: response,
		}
	}

	return nil
}

func (c *httpClient) post(u string, input interface{}, output interface{}, expect ...int) (*http.Response, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(input); err != nil {
		return nil, errors.Wrap(err, "failed to serialize request body")
	}

	req, err := http.NewRequest(http.MethodPost, u, &buf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new HTTP request")
	}

	if err := c.sign(req); err != nil {
		return nil, errors.Wrap(err, "failed to sign HTTP request")
	}
	response, err := c.cl.Do(req)
	if err != nil {
		return nil, errors.Wrapf(ErrRequestFailure, "reason: %s", err)
	}

	return response, c.process(response, output, expect...)
}

func (c *httpClient) put(u string, input interface{}, output interface{}, expect ...int) (*http.Response, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(input); err != nil {
		return nil, errors.Wrap(err, "failed to serialize request body")
	}
	req, err := http.NewRequest(http.MethodPut, u, &buf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build request")
	}

	if err := c.sign(req); err != nil {
		return nil, errors.Wrap(err, "failed to sign HTTP request")
	}

	response, err := c.cl.Do(req)
	if err != nil {
		return nil, errors.Wrapf(ErrRequestFailure, "reason: %s", err)
	}

	return nil, c.process(response, output, expect...)
}

func (c *httpClient) get(u string, query url.Values, output interface{}, expect ...int) (*http.Response, error) {
	if len(query) > 0 {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new HTTP request")
	}

	if err := c.sign(req); err != nil {
		return nil, errors.Wrap(err, "failed to sign HTTP request")
	}

	response, err := c.cl.Do(req)
	if err != nil {
		return nil, errors.Wrapf(ErrRequestFailure, "reason: %s", err)
	}

	return response, c.process(response, output, expect...)
}

func (c *httpClient) delete(u string, query url.Values, output interface{}, expect ...int) (*http.Response, error) {
	if len(query) > 0 {
		u = fmt.Sprintf("%s?%s", u, query.Encode())
	}
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build request")
	}

	if err := c.sign(req); err != nil {
		return nil, errors.Wrap(err, "failed to sign HTTP request")
	}

	response, err := c.cl.Do(req)
	if err != nil {
		return nil, errors.Wrapf(ErrRequestFailure, "reason: %s", err)
	}

	return response, c.process(response, output, expect...)
}

type (
	// Client structure
	Client struct {
		Phonebook Phonebook
		Directory Directory
	}

	// NodeIter iterator over all nodes
	//
	// If the iterator has finished, a nil error and nil node pointer is returned
	NodeIter interface {
		Next() (*Node, error)
	}

	// FarmIter iterator over all farms
	//
	// If the iterator has finished, a nil error and nil farm pointer is returned
	FarmIter interface {
		Next() (*Farm, error)
	}

	// Directory API interface
	Directory interface {
		FarmRegister(farm Farm) (int64, error)
		FarmUpdate(farm Farm) error
		FarmList(tid int64, name string, page *Pager) (farms []Farm, err error)
		FarmGet(id int64) (farm Farm, err error)
		Farms(cacheSize int) FarmIter

		NodeUpdateUptime(id string, uptime uint64) error
		NodeUpdateUsedResources(id string, resources ResourceAmount, workloads WorkloadAmount) error
	}

	// Phonebook interface
	Phonebook interface {
		Create(user User) (int64, error)
		Get(id int64) (User, error)
		GetUserByNameOrEmail(name, email string) (User, error)
		UserExistsByNameOrEmail(name, email string) bool
		UserHasSamePublicKey(u User, ident UserIdentity) bool
	}

	// Identity is used by the client to authenticate to the explorer API
	Identity interface {
		// The unique ID as known by the explorer
		Identity() string
		// PrivateKey used to sign the requests
		PrivateKey() ed25519.PrivateKey
	}

	// Pager for listing
	Pager struct {
		p int
		s int
	}
)

func (p *Pager) apply(v url.Values) {
	if p == nil {
		return
	}

	if p.p < 1 {
		p.p = 1
	}

	if p.s == 0 {
		p.s = 10
	}

	v.Set("page", fmt.Sprint(p.p))
	v.Set("size", fmt.Sprint(p.s))
}

// Page returns a pager
func Page(page, size int) *Pager {
	return &Pager{p: page, s: size}
}

// NewClient creates a new client, if identity is not nil, it will be used
// to authenticate requests against the server
func NewClient(u string, id Identity) (*Client, error) {
	h, err := newHTTPClient(u, id)
	if err != nil {
		return nil, err
	}
	cl := &Client{
		Phonebook: &httpPhonebook{h},
		Directory: &httpDirectory{h},
	}

	return cl, nil
}

// Signer is a utility to easily sign payloads
type Signer struct {
	pair KeyPair
}

// NewSigner create a signer with a seed
func NewSigner(seed []byte) (*Signer, error) {
	pair, err := FromSeed(seed)
	if err != nil {
		return nil, err
	}

	return &Signer{pair: pair}, nil
}

// NewSignerFromFile loads signer from a seed file
func NewSignerFromFile(path string) (*Signer, error) {
	pair, err := LoadKeyPair(path)

	if err != nil {
		return nil, err
	}

	return &Signer{pair: pair}, nil
}

// SignHex like sign, but return message and signature in hex encoded format
func (s *Signer) SignHex(o ...interface{}) (string, string, error) {
	msg, sig, err := s.Sign(o...)
	if err != nil {
		return "", "", err
	}

	return hex.EncodeToString(msg), hex.EncodeToString(sig), nil
}

// Sign constructs a message from all it's arguments then sign it
func (s *Signer) Sign(o ...interface{}) ([]byte, []byte, error) {
	var buf bytes.Buffer
	for _, x := range o {
		switch x := x.(type) {
		case nil:
		case string:
			buf.WriteString(x)
		case fmt.Stringer:
			buf.WriteString(x.String())
		case []byte:
			buf.Write(x)
		case json.RawMessage:
			buf.Write(x)
		case byte:
			buf.WriteString(fmt.Sprint(x))
		// all int types

		case int:
			buf.WriteString(fmt.Sprint(x))
		case int8:
			buf.WriteString(fmt.Sprint(x))
		case int16:
			buf.WriteString(fmt.Sprint(x))
		case int32:
			buf.WriteString(fmt.Sprint(x))
		case int64:
			buf.WriteString(fmt.Sprint(x))
		// all float types
		case float32:
			buf.WriteString(fmt.Sprint(x))
		case float64:
			buf.WriteString(fmt.Sprint(x))
		default:
			return nil, nil, fmt.Errorf("unsupported type")
		}
	}
	msg := buf.Bytes()
	sig, err := Sign(s.pair.PrivateKey, msg)
	return msg, sig, err
}

type (
	httpDirectory struct {
		*httpClient
	}

	httpNodeIter struct {
		cl       *httpDirectory
		proofs   bool
		page     int
		size     int
		cache    []Node
		cacheIdx int
		finished bool
	}

	httpFarmIter struct {
		cl       *httpDirectory
		page     int
		size     int
		cache    []Farm
		cacheIdx int
		finished bool
	}
)

func (d *httpDirectory) FarmRegister(farm Farm) (int64, error) {
	var output struct {
		ID int64 `json:"id"`
	}

	_, err := d.post(d.url("farms"), farm, &output, http.StatusCreated)
	return output.ID, err
}

func (d *httpDirectory) FarmUpdate(farm Farm) error {
	_, err := d.put(d.url("farms", fmt.Sprintf("%d", farm.ID)), farm, nil, http.StatusOK)
	return err
}

func (d *httpDirectory) FarmList(tid int64, name string, page *Pager) (farms []Farm, err error) {
	query := url.Values{}
	page.apply(query)
	if tid > 0 {
		query.Set("owner", fmt.Sprint(tid))
	}
	if len(name) != 0 {
		query.Set("name", name)
	}
	_, err = d.get(d.url("farms"), query, &farms, http.StatusOK)
	return
}

func (d *httpDirectory) FarmGet(id int64) (farm Farm, err error) {
	_, err = d.get(d.url("farms", fmt.Sprint(id)), nil, &farm, http.StatusOK)
	return
}

func (d *httpDirectory) Farms(cacheSize int) FarmIter {
	// pages start at index 1
	return &httpFarmIter{cl: d, size: cacheSize, page: 1}
}

func (fi *httpFarmIter) Next() (*Farm, error) {
	// check if there are still cached farms
	if fi.cacheIdx >= len(fi.cache) {
		if fi.finished {
			return nil, nil
		}
		// pull new data in cache
		pager := Page(fi.page, fi.size)
		farms, err := fi.cl.FarmList(0, "", pager)
		if err != nil {
			return nil, errors.Wrap(err, "could not get farms")
		}
		if len(farms) == 0 {
			// iteration finished, no more  farms
			return nil, nil
		}
		fi.cache = farms
		fi.cacheIdx = 0
		fi.page++
		if len(farms) < fi.size {
			fi.finished = true
		}
	}
	fi.cacheIdx++
	return &fi.cache[fi.cacheIdx-1], nil
}

func (d *httpDirectory) NodeRegister(node Node) error {
	_, err := d.post(d.url("nodes"), node, nil, http.StatusCreated)
	return err
}

// func (d *httpDirectory) NodeList(filter NodeFilter, pager *Pager) (nodes []Node, err error) {
// 	query := url.Values{}
// 	pager.apply(query)
// 	filter.Apply(query)
// 	_, err = d.get(d.url("nodes"), query, &nodes, http.StatusOK)
// 	return
// }

func (d *httpDirectory) NodeGet(id string, proofs bool) (node Node, err error) {
	query := url.Values{}
	query.Set("proofs", fmt.Sprint(proofs))
	_, err = d.get(d.url("nodes", id), query, &node, http.StatusOK)
	return
}

func (d *httpDirectory) NodeSetInterfaces(id string, ifaces []Iface) error {
	_, err := d.post(d.url("nodes", id, "interfaces"), ifaces, nil, http.StatusCreated)
	return err
}

func (d *httpDirectory) NodeSetPorts(id string, ports []uint) error {
	var input struct {
		P []uint `json:"ports"`
	}
	input.P = ports

	_, err := d.post(d.url("nodes", id, "ports"), input, nil, http.StatusOK)
	return err
}

func (d *httpDirectory) NodeSetPublic(id string, pub PublicIface) error {
	_, err := d.post(d.url("nodes", id, "configure_public"), pub, nil, http.StatusCreated)
	return err
}

func (d *httpDirectory) NodeUpdateUptime(id string, uptime uint64) error {
	input := struct {
		U uint64 `json:"uptime"`
	}{
		U: uptime,
	}

	_, err := d.post(d.url("nodes", id, "uptime"), input, nil, http.StatusOK)
	return err
}

func (d *httpDirectory) NodeUpdateUsedResources(id string, resources ResourceAmount, workloads WorkloadAmount) error {
	input := struct {
		ResourceAmount
		WorkloadAmount
	}{
		resources,
		workloads,
	}
	_, err := d.post(d.url("nodes", id, "used_resources"), input, nil, http.StatusOK)
	return err
}

// func (d *httpDirectory) Nodes(cacheSize int, proofs bool) NodeIter {
// 	// pages start at index 1
// 	return &httpNodeIter{cl: d, size: cacheSize, page: 1, proofs: proofs}
// }

// func (ni *httpNodeIter) Next() (*Node, error) {
// 	// check if there are still cached nodes
// 	if ni.cacheIdx >= len(ni.cache) {
// 		if ni.finished {
// 			return nil, nil
// 		}
// 		// pull new data in cache
// 		pager := Page(ni.page, ni.size)
// 		filter := NodeFilter{}.WithProofs(ni.proofs)
// 		nodes, err := ni.cl.NodeList(filter, pager)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "could not get nodes")
// 		}
// 		if len(nodes) == 0 {
// 			// no more nodes, iteration finished
// 			return nil, nil
// 		}
// 		ni.cache = nodes
// 		ni.cacheIdx = 0
// 		ni.page++
// 		if len(nodes) < ni.size {
// 			ni.finished = true
// 		}
// 	}
// 	ni.cacheIdx++
// 	return &ni.cache[ni.cacheIdx-1], nil
// }

type httpPhonebook struct {
	*httpClient
}

func (p *httpPhonebook) Create(user User) (int64, error) {
	var out User
	if _, err := p.post(p.url("users"), user, &out); err != nil {
		return 0, err
	}

	return int64(out.ID), nil
}

func (p *httpPhonebook) List(name, email string, page *Pager) (output []User, err error) {
	query := url.Values{}
	page.apply(query)
	if len(name) != 0 {
		query.Set("name", name)
	}
	if len(email) != 0 {
		query.Set("email", email)
	}

	_, err = p.get(p.url("users"), query, &output, http.StatusOK)

	return
}

func (p *httpPhonebook) Get(id int64) (user User, err error) {
	_, err = p.get(p.url("users", fmt.Sprint(id)), nil, &user, http.StatusOK)
	return
}

func (p *httpPhonebook) GetUserByNameOrEmail(name, email string) (User, error) {
	pager := Page(1, 5)
	u := User{}
	users_list, err := p.List(name, email, pager)
	if err != nil {
		return u, err
	}
	if len(users_list) == 0 {
		return u, errors.New("user doesn't exist")
	}
	return users_list[0], nil

}
func (p *httpPhonebook) UserExistsByNameOrEmail(name, email string) bool {
	_, err := p.GetUserByNameOrEmail(name, email)
	if err != nil {
		return true
	}
	return false

}

func (p *httpPhonebook) UserHasSamePublicKey(u User, ident UserIdentity) bool {
	return hex.EncodeToString(ident.Key().PublicKey) == u.Pubkey
}

// Validate the signature of this message for the user, signature and message are hex encoded
func (p *httpPhonebook) Validate(id int64, message, signature string) (bool, error) {
	var input struct {
		S string `json:"signature"`
		M string `json:"payload"`
	}
	input.S = signature
	input.M = message

	var output struct {
		V bool `json:"is_valid"`
	}

	_, err := p.post(p.url("users", fmt.Sprint(id), "validate"), input, &output, http.StatusOK)
	if err != nil {
		return false, err
	}

	return output.V, nil
}
