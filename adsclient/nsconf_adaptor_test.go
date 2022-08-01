/*
Copyright 2022 Citrix Systems, Inc
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package adsclient

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/citrix/citrix-xds-adaptor/nsconfigengine"
	"github.com/citrix/citrix-xds-adaptor/tests/env"

	netscaler "github.com/citrix/adc-nitro-go/service"
)

func init() {
	env.Init()
}

func verifyFeatures(t *testing.T, client *netscaler.NitroClient, features []string) {
	found := 0
	result, err := client.ListEnabledFeatures()
	if err != nil {
		t.Error("Failed to retrieve features", err)
		log.Println("Cannot continue")
		return
	}
	for _, f := range features {
		for _, r := range result {
			if strings.EqualFold(f, r) {
				found = found + 1
			}
		}
	}
	if found != len(features) {
		t.Error("Requested features do not match enabled features=", features, "result=", result)
	}
}

func verifyModes(t *testing.T, client *netscaler.NitroClient, modes []string) {
	found := 0
	result, err := client.ListEnabledModes()
	if err != nil {
		t.Error("Failed to retrieve modes", err)
		log.Println("Cannot continue")
		return
	}
	for _, m := range modes {
		for _, r := range result {
			if strings.EqualFold(m, r) {
				found = found + 1
			}
		}
	}
	if found != len(modes) {
		t.Error("Requested modes do not match enabled modes=", modes, "result=", result)
	}
}

func Test_bootstrapConfig(t *testing.T) {
	t.Log("Verify BootStrap Config")
	var err interface{}
	os.Setenv("CLUSTER_ID", "Kubernetes")
	os.Setenv("POD_NAMESPACE", "default")
	os.Setenv("APPLICATION_NAME", "myapp")
	os.Setenv("INSTANCE_IP", env.GetNetscalerIP())
	analyticserverip := "1.1.1.1"
	licenseserverip := "1.1.1.2"
	configAdaptorerrorlog := "Unable to get a config adaptor. newConfigAdaptor failed with"
	bootstrapconfiglog := "Config verification failed for sidecar bootstrap config, error"
	nsinfo := new(NSDetails)
	nsinfo.NetscalerURL = env.GetNetscalerURL()
	nsinfo.NetscalerUsername = env.GetNetscalerUser()
	nsinfo.NetscalerPassword = env.GetNetscalerPassword()
	nsinfo.NetscalerVIP = ""
	nsinfo.NetProfile = "k8s"
	nsinfo.AnalyticsServerIP = analyticserverip
	nsinfo.LicenseServer = licenseserverip
	nsinfo.LogProxyURL = "ns-logproxy.citrix-system"
	nsinfo.adsServerPort = "15010"
	nsinfo.bootStrapConfReqd = true
	multiClusterIngress = true
	multiClusterPolExprStr = ".global"
	multiClusterListenPort = 15443
	configAd, err := newConfigAdaptor(nsinfo)
	if err != nil {
		t.Errorf(" %s  %v", configAdaptorerrorlog, err)
	}
	relNo, _ := nsconfigengine.GetNsReleaseBuild()
	if strings.Contains(env.GetNetscalerURL(), "localhost") || strings.Contains(env.GetNetscalerURL(), "127.0.0.1") {
		t.Logf("Verifying sidecar bootstrap config")
		configs := []env.VerifyNitroConfig{
			{"service", "dns_service", map[string]interface{}{"name": "dns_service", "port": 53, "servicetype": "DNS", "healthmonitor": "NO"}},
			{"lbvserver", "dns_vserver", map[string]interface{}{"name": "dns_vserver", "servicetype": "DNS"}},
			{netscaler.Lbvserver_service_binding.Type(), "dns_vserver", map[string]interface{}{"name": "dns_vserver", "servicename": "dns_service"}},
			{"dnsnameserver", "dns_vserver", map[string]interface{}{"dnsvservername": "dns_vserver"}},
			{"nsacl", "allowpromexp", map[string]interface{}{"aclname": "allowpromexp", "aclaction": "ALLOW", "protocol": "TCP", "destportval": "8888", "priority": 65536, "kernelstate": "APPLIED"}},
			{"nsacl", "denyall", map[string]interface{}{"aclname": "denyall", "aclaction": "DENY", "priority": 100000, "kernelstate": "APPLIED"}},
		}
		configs2 := []env.VerifyNitroConfig{
			{"nsacl", "allowadmserver", map[string]interface{}{"aclname": "allowadmserver", "aclaction": "ALLOW", "srcipval": "1.1.1.1", "priority": 65537}},
			{"nsacl", "allowlicenseserver", map[string]interface{}{"aclname": "allowlicenseserver", "aclaction": "ALLOW", "srcipval": "1.1.1.2", "priority": 65538}},
			{"nsacl", "allownitro", map[string]interface{}{"aclname": "allownitro", "aclaction": "ALLOW", "protocol": "TCP", "destportval": "9443", "priority": 65540, "kernelstate": "APPLIED"}},
			{"nsacl", "allowicmp", map[string]interface{}{"aclname": "allowicmp", "aclaction": "ALLOW", "protocol": "ICMP", "priority": 65546, "kernelstate": "APPLIED"}},
			{"lbvserver", "drop_all_vserver", map[string]interface{}{"name": "drop_all_vserver", "servicetype": "ANY", "ipv46": "*", "port": 65535, "listenpolicy": "CLIENT.TCP.DSTPORT.NE(15010) && CLIENT.IP.DST.NE(1.1.1.1) && CLIENT.TCP.DSTPORT.NE(5557) && CLIENT.TCP.DSTPORT.NE(5558) && CLIENT.TCP.DSTPORT.NE(5563) && CLIENT.IP.DST.NE(1.1.1.2) && CLIENT.TCP.DSTPORT.NE(27000) && CLIENT.TCP.DSTPORT.NE(7279)"}},
		}
		endpointConfig := []env.VerifyNitroConfig{}
		// Label Endpoints support is provided from 13.1 version
		if relNo >= 13.1 {
			metaData := "Kubernetes." + os.Getenv("POD_NAMESPACE") + "." + os.Getenv("APPLICATION_NAME") + ".*" + ".*" + ".*"
			endpointConfig = []env.VerifyNitroConfig{
				//{"endpointinfo", os.Getenv("INSTANCE_IP"), map[string]interface{}{"endpointkind": "IP", "endpointname": os.Getenv("INSTANCE_IP"), "endpointmetadata": metaData}},
				{"endpointinfo", nsLoopbackIP, map[string]interface{}{"endpointkind": "IP", "endpointname": nsLoopbackIP, "endpointmetadata": metaData}},
				{"endpointinfo", "192.0.0.1", map[string]interface{}{"endpointkind": "IP", "endpointname": "192.0.0.1", "endpointmetadata": metaData}},
			}
			configs = append(configs, endpointConfig...)
		}
		configs3 := []env.VerifyNitroConfig{}
		configs3 = append(configs, configs2...)
		err = env.VerifyConfigBlockPresence(configAd.client, configs3)
		if err != nil {
			t.Errorf("%s %v", bootstrapconfiglog, err)
		}
		nsinfo.AnalyticsServerIP = ""
		nsinfo.LicenseServer = licenseserverip
		nsinfo.caServerPort = "15012"
		nsinfo.bootStrapConfReqd = true
		configAd, err := newConfigAdaptor(nsinfo)
		if err != nil {
			t.Errorf("%s %v", configAdaptorerrorlog, err)
		}
		configs2 = []env.VerifyNitroConfig{
			{"nsacl", "allowlicenseserver", map[string]interface{}{"aclname": "allowlicenseserver", "aclaction": "ALLOW", "srcipval": "1.1.1.2", "priority": 65538}},
			{"lbvserver", "drop_all_vserver", map[string]interface{}{"name": "drop_all_vserver", "servicetype": "ANY", "ipv46": "*", "port": 65535, "listenpolicy": "CLIENT.TCP.DSTPORT.NE(15010) && CLIENT.TCP.DSTPORT.NE(15012) && CLIENT.IP.DST.NE(1.1.1.2) && CLIENT.TCP.DSTPORT.NE(27000) && CLIENT.TCP.DSTPORT.NE(7279)"}},
		}
		configs3 = append(configs, configs2...)
		err = env.VerifyConfigBlockPresence(configAd.client, configs3)
		if err != nil {
			t.Errorf("%s %v", bootstrapconfiglog, err)
		}
		nsinfo.AnalyticsServerIP = analyticserverip
		nsinfo.LicenseServer = ""
		nsinfo.caServerPort = ""
		nsinfo.bootStrapConfReqd = true
		configAd, err = newConfigAdaptor(nsinfo)
		if err != nil {
			t.Errorf("%s %v", configAdaptorerrorlog, err)
		}
		configs2 = []env.VerifyNitroConfig{
			{"nsacl", "allowadmserver", map[string]interface{}{"aclname": "allowadmserver", "aclaction": "ALLOW", "srcipval": "1.1.1.1", "priority": 65537}},
			{"lbvserver", "drop_all_vserver", map[string]interface{}{"name": "drop_all_vserver", "servicetype": "ANY", "ipv46": "*", "port": 65535, "listenpolicy": "CLIENT.TCP.DSTPORT.NE(15010) && CLIENT.IP.DST.NE(1.1.1.1) && CLIENT.TCP.DSTPORT.NE(5557) && CLIENT.TCP.DSTPORT.NE(5558) && CLIENT.TCP.DSTPORT.NE(5563)"}},
		}
		configs3 = append(configs, configs2...)
		err = env.VerifyConfigBlockPresence(configAd.client, configs3)
		if err != nil {
			t.Errorf("%s %v", bootstrapconfiglog, err)
		}
		nsinfo.LicenseServer = analyticserverip
		nsinfo.bootStrapConfReqd = true
		configAd, err = newConfigAdaptor(nsinfo)
		if err != nil {
			t.Errorf("%s %v", configAdaptorerrorlog, err)
		}
		err = env.VerifyConfigBlockPresence(configAd.client, configs3)
		if err != nil {
			t.Errorf("%s %v", bootstrapconfiglog, err)
		}
		nsinfo.AnalyticsServerIP = ""
		nsinfo.LicenseServer = ""
		nsinfo.bootStrapConfReqd = true
		configAd, err = newConfigAdaptor(nsinfo)
		if err != nil {
			t.Errorf("%s %v", configAdaptorerrorlog, err)
		}
		configs2 = []env.VerifyNitroConfig{
			{"lbvserver", "drop_all_vserver", map[string]interface{}{"name": "drop_all_vserver", "servicetype": "ANY", "ipv46": "*", "port": 65535, "listenpolicy": "CLIENT.TCP.DSTPORT.NE(15010)"}},
		}
		configs3 = append(configs, configs2...)
		err = env.VerifyConfigBlockPresence(configAd.client, configs3)
		if err != nil {
			t.Errorf("%s %v", bootstrapconfiglog, err)
		}
	}
	t.Log("Verify Features Applied")
	features := []string{"SSL", "LB", "CS", "REWRITE", "RESPONDER", "APPFLOW"}
	verifyFeatures(t, configAd.client, features)
	t.Log("Verify Modes Applied")
	modes := []string{"ULFD"}
	verifyModes(t, configAd.client, modes)
	t.Log("Verify bootstrap config")
	configs := []env.VerifyNitroConfig{
		{netscaler.Nstcpprofile.Type(), "nstcp_default_profile", map[string]interface{}{"name": "nstcp_default_profile", "mss": 1410}},
		{netscaler.Nstcpprofile.Type(), "nstcp_internal_apps", map[string]interface{}{"name": "nstcp_internal_apps", "mss": 1410}},
		{netscaler.Nstcpprofile.Type(), "nsulfd_default_profile", map[string]interface{}{"name": "nsulfd_default_profile", "mss": 1410}},
		{netscaler.Nshttpprofile.Type(), "nshttp_default_profile", map[string]interface{}{"name": "nshttp_default_profile", "http2": "ENABLED", "http2maxconcurrentstreams": 1000}},
		{netscaler.Responderaction.Type(), "return404", map[string]interface{}{"name": "return404", "type": "respondwith", "target": "\"HTTP/1.1 404 Not found\r\n\r\n\""}},
		{netscaler.Responderpolicy.Type(), "return404", map[string]interface{}{"name": "return404", "rule": "true", "action": "return404"}},
		{netscaler.Lbvserver.Type(), "ns_blackhole_http", map[string]interface{}{"name": "ns_blackhole_http", "servicetype": "HTTP"}},
		{netscaler.Service.Type(), "ns_blackhole_http", map[string]interface{}{"name": "ns_blackhole_http", "servername": "127.0.0.1", "port": 1, "servicetype": "HTTP", "healthmonitor": "NO"}},
		{netscaler.Lbvserver_service_binding.Type(), "ns_blackhole_http", map[string]interface{}{"name": "ns_blackhole_http", "servicename": "ns_blackhole_http"}},
		{netscaler.Lbvserver_responderpolicy_binding.Type(), "ns_blackhole_http", map[string]interface{}{"name": "ns_blackhole_http", "policyname": "return404", "priority": 1}},
		{netscaler.Lbvserver.Type(), "ns_dummy_http", map[string]interface{}{"name": "ns_dummy_http", "servicetype": "HTTP"}},
		{netscaler.Lbvserver_service_binding.Type(), "ns_dummy_http", map[string]interface{}{"name": "ns_dummy_http", "servicename": "ns_blackhole_http"}},
	}

	if relNo >= 13.1 {
		metaData := os.Getenv("CLUSTER_ID") + "." + os.Getenv("POD_NAMESPACE") + "." + os.Getenv("APPLICATION_NAME") + ".*" + ".*" + ".*"
		//os.Getenv("CLUSTER_ID") + os.Getenv("POD_NAMESPACE") + ".service." + os.Getenv("APPLICATION_NAME")
		epResIP := os.Getenv("INSTANCE_IP")
		if len(configAd.vserverIP) > 0 { // If gateway mode, then add endpoint info for VIP
			epResIP = configAd.vserverIP
		}
		endpointConfig := env.VerifyNitroConfig{
			"endpointinfo", epResIP, map[string]interface{}{"endpointkind": "IP", "endpointname": epResIP, "endpointmetadata": metaData},
		}
		configs = append(configs, endpointConfig)
	}

	err = env.VerifyConfigBlockPresence(configAd.client, configs)
	if err != nil {
		t.Errorf("Config verification failed for bootstrap config, error %v", err)
	}
	t.Log("Verify logproxy/appflow related config")
	configs = []env.VerifyNitroConfig{
		{netscaler.Appflowparam.Type(), "", map[string]interface{}{"templaterefresh": 60, "securityinsightrecordinterval": 60, "httpurl": "ENABLED", "httpcookie": "ENABLED", "httpreferer": "ENABLED", "httpmethod": "ENABLED", "httphost": "ENABLED", "httpuseragent": "ENABLED", "httpcontenttype": "ENABLED", "securityinsighttraffic": "ENABLED", "httpquerywithurl": "ENABLED", "urlcategory": "ENABLED", "distributedtracing": "ENABLED", "disttracingsamplingrate": 100}},
		{"analyticsprofile", "ns_analytics_default_http_profile", map[string]interface{}{"name": "ns_analytics_default_http_profile", "type": "webinsight", "httpurl": "ENABLED", "httpmethod": "ENABLED", "httphost": "ENABLED", "httpuseragent": "ENABLED", "urlcategory": "ENABLED", "httpcontenttype": "ENABLED", "httpvia": "ENABLED", "httpdomainname": "ENABLED", "httpurlquery": "ENABLED"}},
		{"analyticsprofile", "ns_analytics_default_tcp_profile", map[string]interface{}{"name": "ns_analytics_default_tcp_profile", "type": "tcpinsight"}},
	}
	err = env.VerifyConfigBlockPresence(configAd.client, configs)
	if err != nil {
		t.Errorf("Config verification failed for logproxy config, error %v", err)
	}
}

func Test_getBoolEnv(t *testing.T) {
	os.Setenv("EMPTY_VAR", "")
	os.Setenv("TRUE_VAR", "1")
	os.Setenv("FALSE_VAR", "f")
	os.Setenv("INVALID_VAR", "helloworld")

	tc := map[string]struct {
		envVar    string
		expOutput bool
	}{
		"empty-env": {
			envVar:    "EMPTY_VAR",
			expOutput: false,
		},
		"true-env": {
			envVar:    "TRUE_VAR",
			expOutput: true,
		},
		"false-env": {
			envVar:    "FALSE_VAR",
			expOutput: false,
		},
		"invalid-var": {
			envVar:    "INVALID_VAR",
			expOutput: false,
		},
	}

	for id, c := range tc {
		if c.expOutput != getBoolEnv(c.envVar) {
			t.Errorf("Failed for %s", id)
		} else {
			t.Logf("Succeed for %s", id)
		}
	}
}

//func getIntEnv(key string) int
func Test_getIntEnv(t *testing.T) {
	os.Setenv("EMPTY_VAR", "")
	os.Setenv("VALID_VAR", "10001")
	os.Setenv("INVALID_VAR", "helloworld")

	tc := map[string]struct {
		envVar    string
		expOutput int
	}{
		"empty-env": {
			envVar:    "EMPTY_VAR",
			expOutput: -1,
		},
		"valid-env": {
			envVar:    "VALID_VAR",
			expOutput: 10001,
		},
		"invalid-var": {
			envVar:    "INVALID_VAR",
			expOutput: -1,
		},
	}

	for id, c := range tc {
		if c.expOutput != getIntEnv(c.envVar) {
			t.Errorf("Failed for %s", id)
		} else {
			t.Logf("Succeed for %s", id)
		}
	}
}
