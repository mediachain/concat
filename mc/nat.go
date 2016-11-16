package mc

import (
	"errors"
	"fmt"
	multiaddr "github.com/multiformats/go-multiaddr"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var (
	BadAddressSpec = errors.New("Bad NAT address specification")
)

type NATConfig struct {
	Opt  int
	spec string              // address spec when option = NATConfigManual
	addr multiaddr.Multiaddr // public address when option = NATConfigManual
}

const (
	NATConfigNone = iota
	NATConfigAuto
	NATConfigManual
)

var natConfigString = []string{"none", "auto", "manual"}

func (cfg *NATConfig) String() string {
	switch cfg.Opt {
	case NATConfigManual:
		return cfg.spec
	default:
		return natConfigString[cfg.Opt]
	}
}

func (cfg *NATConfig) PublicAddr(base multiaddr.Multiaddr) (multiaddr.Multiaddr, error) {
	if cfg.Opt != NATConfigManual {
		return nil, BadAddressSpec
	}

	if cfg.addr != nil {
		return cfg.addr, nil
	}

	ix := strings.LastIndex(cfg.spec, ":")
	switch {
	case ix < 0:
		addr, err := GetPublicIP()
		if err != nil {
			return nil, err
		}

		port, err := base.ValueForProtocol(multiaddr.P_TCP)
		if err != nil {
			return nil, err
		}

		return cfg.makePublicAddr(addr, port)

	case cfg.spec[:ix] == "*":
		addr, err := GetPublicIP()
		if err != nil {
			return nil, err
		}
		port := cfg.spec[ix+1:]
		return cfg.makePublicAddr(addr, port)

	default:
		addr, port := cfg.spec[:ix], cfg.spec[ix+1:]
		return cfg.makePublicAddr(addr, port)
	}
}

func (cfg *NATConfig) makePublicAddr(addr, port string) (multiaddr.Multiaddr, error) {
	maddr, err := ParseAddress(fmt.Sprintf("/ip4/%s/tcp/%s", addr, port))
	if err != nil {
		return nil, err
	}

	cfg.addr = maddr
	return maddr, nil
}

func (cfg *NATConfig) Clear() {
	cfg.addr = nil
}

var natcfgrx *regexp.Regexp

func init() {
	rx, err := regexp.Compile("([*])|([*]:[0-9]+)|([0-9]{1,3}([.][0-9]{1,3}){3}:[0-9]+)")
	if err != nil {
		log.Fatal(err)
	}
	natcfgrx = rx
}

func NATConfigFromString(str string) (cfg NATConfig, err error) {
	switch str {
	case "none":
		cfg.Opt = NATConfigNone
		return cfg, nil
	case "auto":
		cfg.Opt = NATConfigAuto
		return cfg, nil
	default:
		if !natcfgrx.Match([]byte(str)) {
			return cfg, BadAddressSpec
		}
		cfg.Opt = NATConfigManual
		cfg.spec = str
		return cfg, nil
	}
}

func GetPublicIP() (string, error) {
	res, err := http.Get("http://ifconfig.co/ip")
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
