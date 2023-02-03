package amcrest

import (
	"fmt"
	"ha-adapters/pkg/parsers"
	"strings"
)

func (s *AmcrestDevice) GetConfig() (map[string]string, error) {
	info, err := s.request("/cgi-bin/configManager.cgi?action=getConfig&name=All")
	if err != nil {
		return nil, err
	}
	configs := parsers.ParseManyKV(info, "\n")

	// Keys all seem to contain "table.All" unncessarily, but setting config doesn't
	// so try to remove
	ret := make(map[string]string)
	for k, v := range configs {
		k = strings.TrimPrefix(k, "table.All.")
		ret[k] = v
	}

	return ret, nil
}

func (s *AmcrestDevice) SetConfig(kv ...string) error {
	if len(kv)%2 != 0 {
		panic("Expected even pairs")
	}
	url := "/cgi-bin/configManager.cgi?action=setConfig"
	for i := 0; i < len(kv); i += 2 {
		url += fmt.Sprintf("&%s=%s", kv[i], kv[i+1])
	}
	_, err := s.request(url)
	return err
}

func (s *AmcrestDevice) SetLight(on bool) error {
	if on {
		return s.SetConfig("Lighting_V2[0][0][1].Mode", "ForceOn", "Lighting_V2[0][0][1].State", "On")
	} else { // auto
		return s.SetConfig("Lighting_V2[0][0][1].Mode", "Auto", "Lighting_V2[0][0][1].State", "Flicker")
	}
}
