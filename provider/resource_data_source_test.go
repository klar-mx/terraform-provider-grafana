package provider

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSource_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var dataSource gapi.DataSource

	var resourceTests = []struct {
		resource         string
		config           string
		attrChecks       map[string]string
		additionalChecks []resource.TestCheckFunc
		verifyImport     bool // Only test import when `json_data_encoded` is set. Data sources with `json_data` cannot have their data imported.
	}{
		{
			resource: "grafana_data_source.loki",
			config: `
		resource "grafana_data_source" "tempo" {
			name = "Tempo Linked to Loki"
			type = "tempo"
		}

		resource "grafana_data_source" "loki" {
			type                = "loki"
			name                = "loki"
			url                 = "http://acc-test.invalid/"
			json_data {
				max_lines = 2022

				derived_field {
					name = "WithoutDatasource"
					matcher_regex = "(?:traceID|trace_id)=(\\w+)"
					url = "example.com/$${__value.raw}"
				}

				derived_field {
					name = "WithDatasource"
					matcher_regex = "(?:traceID|trace_id)=(\\w+)"
					url = "$${__value.raw}"
					datasource_uid = grafana_data_source.tempo.uid
				}
			}
		}
		`,
			attrChecks: map[string]string{
				"type":                             "loki",
				"name":                             "loki",
				"url":                              "http://acc-test.invalid/",
				"json_data.0.max_lines":            "2022",
				"json_data.0.derived_field.0.name": "WithoutDatasource",
				"json_data.0.derived_field.0.matcher_regex": "(?:traceID|trace_id)=(\\w+)",
				"json_data.0.derived_field.0.url":           "example.com/${__value.raw}",
				"json_data.0.derived_field.1.name":          "WithDatasource",
				"json_data.0.derived_field.1.matcher_regex": "(?:traceID|trace_id)=(\\w+)",
				"json_data.0.derived_field.1.url":           "${__value.raw}",
			},
			additionalChecks: []resource.TestCheckFunc{
				func(s *terraform.State) error {
					// Check datasource IDs
					derivedFields := dataSource.JSONData["derivedFields"].([]interface{})
					if len(derivedFields) != 2 {
						return fmt.Errorf("expected 2 derived fields, got %d", len(derivedFields))
					}
					firstDerivedField := derivedFields[0].(map[string]interface{})
					if _, ok := firstDerivedField["datasourceUid"]; ok {
						return fmt.Errorf("expected empty datasource_uid")
					}
					secondDerivedField := derivedFields[1].(map[string]interface{})
					if !uidRegexp.MatchString(secondDerivedField["datasourceUid"].(string)) {
						return fmt.Errorf("expected valid datasource_uid")
					}
					return nil
				},
			},
		},
		{
			resource: "grafana_data_source.testdata",
			config: `
		resource "grafana_data_source" "testdata" {
			type                = "testdata"
			name                = "testdata"
			access_mode					= "direct"
			basic_auth_enabled  = true
			basic_auth_password = "ba_password"
			basic_auth_username = "ba_username"
			database_name       = "db_name"
			is_default					= true
			url                 = "http://acc-test.invalid/"
			username            = "user"
			password            = "pass"
		}
		`,
			attrChecks: map[string]string{
				"type":                "testdata",
				"name":                "testdata",
				"access_mode":         "direct",
				"basic_auth_enabled":  "true",
				"basic_auth_password": "ba_password",
				"basic_auth_username": "ba_username",
				"database_name":       "db_name",
				"is_default":          "true",
				"url":                 "http://acc-test.invalid/",
				"username":            "user",
				"password":            "pass",
			},
		},
		{
			resource: "grafana_data_source.graphite",
			config: `
			resource "grafana_data_source" "graphite" {
				type = "graphite"
				name = "graphite"
				url  = "http://acc-test.invalid/"
				json_data {
					graphite_version = "1.1"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                         "graphite",
				"name":                         "graphite",
				"url":                          "http://acc-test.invalid/",
				"json_data.0.graphite_version": "1.1",
			},
		},
		{
			resource: "grafana_data_source.influx",
			config: `
			resource "grafana_data_source" "influx" {
				type          = "influx"
				name          = "influx"
				database_name = "db_name"
				username      = "user"
				password      = "pass"
				url           = "http://acc-test.invalid/"
				json_data {
					time_interval = "60s"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                      "influx",
				"name":                      "influx",
				"database_name":             "db_name",
				"username":                  "user",
				"password":                  "pass",
				"url":                       "http://acc-test.invalid/",
				"json_data.0.time_interval": "60s",
			},
		},
		{
			resource: "grafana_data_source.influx",
			config: `
			resource "grafana_data_source" "influx" {
				type         = "influxdb"
				name         = "influx"
				url          = "http://acc-test.invalid/"
			    http_headers = {
				    Authorization = "Token sdkfjsdjflkdsjflksjdklfjslkdfjdksljfldksjsflkj"
			    }
				json_data {
					default_bucket        = "telegraf"
					organization          = "organization"
					tls_auth              = false
					tls_auth_with_ca_cert = false
					version               = "Flux"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                              "influxdb",
				"name":                              "influx",
				"url":                               "http://acc-test.invalid/",
				"json_data.0.default_bucket":        "telegraf",
				"json_data.0.organization":          "organization",
				"json_data.0.tls_auth":              "false",
				"json_data.0.tls_auth_with_ca_cert": "false",
				"json_data.0.version":               "Flux",
				"http_headers.Authorization":        "Token sdkfjsdjflkdsjflksjdklfjslkdfjdksljfldksjsflkj",
			},
			additionalChecks: []resource.TestCheckFunc{
				func(s *terraform.State) error {
					if dataSource.Name != "influx" {
						return fmt.Errorf("bad name: %s", dataSource.Name)
					}
					if v, ok := dataSource.JSONData["httpHeaderName1"]; !ok && v != "Authorization" {
						return fmt.Errorf("http header Authorization not found")
					}
					return nil
				},
			},
		},
		{
			resource: "grafana_data_source.influx-arbitrary",
			config: `
			resource "grafana_data_source" "influx-arbitrary" {
				type         = "influxdb"
				name         = "influx"
				url          = "http://acc-test.invalid/"
			    http_headers = {
				    Authorization = "Token sdkfjsdjflkdsjflksjdklfjslkdfjdksljfldksjsflkj"
			    }
				json_data_encoded = jsonencode({
					defaultBucket         = "telegraf"
					organization          = "organization"
					tlsAuth               = false
					tlsAuthWithCACert     = false
					version               = "Flux"
				})
			}
			`,
			attrChecks: map[string]string{
				"type":                       "influxdb",
				"name":                       "influx",
				"url":                        "http://acc-test.invalid/",
				"json_data_encoded":          `{"defaultBucket":"telegraf","organization":"organization","tlsAuth":false,"tlsAuthWithCACert":false,"version":"Flux"}`,
				"http_headers.Authorization": "Token sdkfjsdjflkdsjflksjdklfjslkdfjdksljfldksjsflkj",
			},
			additionalChecks: []resource.TestCheckFunc{
				func(s *terraform.State) error {
					if dataSource.Name != "influx" {
						return fmt.Errorf("bad name: %s", dataSource.Name)
					}
					expected := map[string]interface{}{
						"defaultBucket":     "telegraf",
						"organization":      "organization",
						"tlsAuth":           false,
						"tlsAuthWithCACert": false,
						"version":           "Flux",
						"httpHeaderName1":   "Authorization",
					}
					if !reflect.DeepEqual(dataSource.JSONData, expected) {
						return fmt.Errorf("bad json_data_encoded: %#v. Expected: %+v", dataSource.JSONData, expected)
					}
					return nil
				},
			},
			verifyImport: true,
		},
		{
			resource: "grafana_data_source.elasticsearch",
			config: `
			resource "grafana_data_source" "elasticsearch" {
				type          = "elasticsearch"
				name          = "elasticsearch"
				database_name = "[filebeat-]YYYY.MM.DD"
				url 	        = "http://acc-test.invalid/"
				json_data {
					es_version        = "7.0.0"
					interval          = "Daily"
					time_field        = "@timestamp"
					log_message_field = "message"
					log_level_field   = "fields.level"
					max_concurrent_shard_requests = 8
					xpack_enabled     = true
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                          "elasticsearch",
				"name":                          "elasticsearch",
				"database_name":                 "[filebeat-]YYYY.MM.DD",
				"url":                           "http://acc-test.invalid/",
				"json_data.0.es_version":        "7.0.0",
				"json_data.0.interval":          "Daily",
				"json_data.0.time_field":        "@timestamp",
				"json_data.0.log_message_field": "message",
				"json_data.0.log_level_field":   "fields.level",
				"json_data.0.max_concurrent_shard_requests": "8",
			},
			additionalChecks: []resource.TestCheckFunc{
				func(s *terraform.State) error {
					if dataSource.Name != "elasticsearch" {
						return fmt.Errorf("bad name: %s", dataSource.Name)
					}
					if dataSource.JSONData["xpack"].(bool) != true {
						return errors.New("xpack_enabled should be true")
					}
					return nil
				},
			},
		},
		{
			resource: "grafana_data_source.elasticsearch-arbitrary",
			config: `
			resource "grafana_data_source" "elasticsearch-arbitrary" {
				type          = "elasticsearch"
				name          = "elasticsearch-arbitrary"
				database_name = "[filebeat-]YYYY.MM.DD"
				url 	        = "http://acc-test.invalid/"
				json_data_encoded = jsonencode({
					esVersion        = "7.0.0"
					interval          = "Daily"
					timeField        = "@timestamp"
					logMessageField = "message"
					logLevelField   = "fields.level"
					maxConcurrentShardRequests = 8
					xpack     = true
				})
			}
			`,
			attrChecks: map[string]string{
				"type":              "elasticsearch",
				"name":              "elasticsearch-arbitrary",
				"database_name":     "[filebeat-]YYYY.MM.DD",
				"url":               "http://acc-test.invalid/",
				"json_data_encoded": `{"esVersion":"7.0.0","interval":"Daily","logLevelField":"fields.level","logMessageField":"message","maxConcurrentShardRequests":8,"timeField":"@timestamp","xpack":true}`,
			},
			additionalChecks: []resource.TestCheckFunc{
				func(s *terraform.State) error {
					if dataSource.Name != "elasticsearch-arbitrary" {
						return fmt.Errorf("bad name: %s", dataSource.Name)
					}
					if dataSource.JSONData["xpack"].(bool) != true {
						return errors.New("xpack should be true")
					}
					return nil
				},
			},
			verifyImport: true,
		},
		{
			resource: "grafana_data_source.opentsdb",
			config: `
			resource "grafana_data_source" "opentsdb" {
				type = "opentsdb"
				name = "opentsdb"
				url	 = "http://acc-test.invalid/"
				json_data {
					tsdb_resolution = 1
					tsdb_version    = 1
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                        "opentsdb",
				"name":                        "opentsdb",
				"url":                         "http://acc-test.invalid/",
				"json_data.0.tsdb_resolution": "1",
				"json_data.0.tsdb_version":    "1",
			},
		},
		{
			resource: "grafana_data_source.cloudwatch",
			config: `
			resource "grafana_data_source" "cloudwatch" {
				type = "cloudwatch"
				name = "cloudwatch"
				json_data {
					default_region            = "us-east-1"
					auth_type                 = "keys"
					assume_role_arn           = "arn:aws:sts::*:assumed-role/*/*"
					custom_metrics_namespaces = "foo"
					external_id               = "abc123"
					tracing_datasource_uid    = "my-datasource-uid"
				}
				secure_json_data {
					access_key = "123"
					secret_key = "456"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                                  "cloudwatch",
				"name":                                  "cloudwatch",
				"json_data.0.default_region":            "us-east-1",
				"json_data.0.auth_type":                 "keys",
				"json_data.0.assume_role_arn":           "arn:aws:sts::*:assumed-role/*/*",
				"json_data.0.custom_metrics_namespaces": "foo",
				"json_data.0.external_id":               "abc123",
				"secure_json_data.0.access_key":         "123",
				"secure_json_data.0.secret_key":         "456",
			},
			additionalChecks: []resource.TestCheckFunc{
				func(s *terraform.State) error {
					if dataSource.Name != "cloudwatch" {
						return fmt.Errorf("bad name: %s", dataSource.Name)
					}
					datasourceUID := dataSource.JSONData["tracingDatasourceUid"].(string)
					if datasourceUID != "my-datasource-uid" {
						return fmt.Errorf("bad tracing_datasource_uid: %s", datasourceUID)
					}
					return nil
				},
			},
		},
		{
			resource: "grafana_data_source.mssql",
			config: `
				resource "grafana_data_source" "mssql" {
					type          = "mssql"
					name          = "mssql"
					database_name = "db"
					url 	        = "acc-test.invalid:1433"
					json_data {
						max_open_conns    = 0
						max_idle_conns    = 2
						conn_max_lifetime = 14400
						encrypt           = "yes"
					}
					secure_json_data {
						password = "pass"
					}
				}
				`,
			attrChecks: map[string]string{
				"type":                          "mssql",
				"name":                          "mssql",
				"database_name":                 "db",
				"url":                           "acc-test.invalid:1433",
				"json_data.0.max_open_conns":    "0",
				"json_data.0.max_idle_conns":    "2",
				"json_data.0.conn_max_lifetime": "14400",
				"json_data.0.encrypt":           "yes",
				"secure_json_data.0.password":   "pass",
			},
		},
		{
			resource: "grafana_data_source.postgres",
			config: `
				resource "grafana_data_source" "postgres" {
					type          = "postgres"
					name          = "postgres"
					database_name = "db"
					url 	        = "acc-test.invalid:5432"
					username      = "user"
					json_data {
						max_open_conns    = 0
						max_idle_conns    = 2
						conn_max_lifetime = 14400
						postgres_version  = 905
						timescaledb 			= false
					}
					secure_json_data {
						password = "pass"
					}
				}
				`,
			attrChecks: map[string]string{
				"type":                          "postgres",
				"name":                          "postgres",
				"database_name":                 "db",
				"url":                           "acc-test.invalid:5432",
				"json_data.0.max_open_conns":    "0",
				"json_data.0.max_idle_conns":    "2",
				"json_data.0.conn_max_lifetime": "14400",
				"json_data.0.postgres_version":  "905",
				"json_data.0.timescaledb":       "false",
				"secure_json_data.0.password":   "pass",
			},
		},
		{
			resource: "grafana_data_source.prometheus",
			config: `
			resource "grafana_data_source" "prometheus" {
				type = "prometheus"
				name = "prometheus"
				url  = "http://acc-test.invalid:9090"
				json_data {
					http_method = "GET"
					query_timeout = "1"
					sigv4_auth   = true
					sigv4_auth_type = "default"
					sigv4_region    = "eu-west-1"
					manage_alerts   = true
					alertmanager_uid = "Alertmanager"
				}

				http_headers = {
					"header1" = "value1"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                         "prometheus",
				"name":                         "prometheus",
				"url":                          "http://acc-test.invalid:9090",
				"json_data.0.http_method":      "GET",
				"json_data.0.query_timeout":    "1",
				"json_data.0.sigv4_auth":       "true",
				"json_data.0.sigv4_auth_type":  "default",
				"json_data.0.sigv4_region":     "eu-west-1",
				"http_headers.header1":         "value1",
				"json_data.0.alertmanager_uid": "Alertmanager",
			},
			additionalChecks: []resource.TestCheckFunc{
				func(s *terraform.State) error {
					if dataSource.Name != "prometheus" {
						return fmt.Errorf("bad name: %s", dataSource.Name)
					}
					if v, ok := dataSource.JSONData["httpHeaderName1"]; !ok && v != "header1" {
						return fmt.Errorf("http header header1 not found")
					}
					if dataSource.JSONData["manageAlerts"].(bool) != true {
						return errors.New("expected manage_alerts to be true")
					}
					return nil
				},
			},
		},
		{
			resource: "grafana_data_source.sentry",
			config: `
			resource "grafana_data_source" "sentry" {
			    type = "grafana-sentry-datasource"
			    name = "sentry"
		        url  = "https://sentry.io"
			    json_data {
			        org_slug = "grafanalabs"
			    }
			    secure_json_data {
			        auth_token = "abc123"
			    }
			}
			`,
			attrChecks: map[string]string{
				"type":                          "grafana-sentry-datasource",
				"name":                          "sentry",
				"url":                           "https://sentry.io",
				"json_data.0.org_slug":          "grafanalabs",
				"secure_json_data.0.auth_token": "abc123",
			},
		},
		{
			resource: "grafana_data_source.stackdriver",
			config: `
			resource "grafana_data_source" "stackdriver" {
				type = "stackdriver"
				name = "stackdriver"
				json_data {
					token_uri = "https://oauth2.googleapis.com/token"
					authentication_type = "jwt"
					default_project = "default-project"
					client_email = "client-email@default-project.iam.gserviceaccount.com"
				}
				secure_json_data {
					private_key = "-----BEGIN PRIVATE KEY-----\nprivate-key\n-----END PRIVATE KEY-----\n"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                            "stackdriver",
				"name":                            "stackdriver",
				"json_data.0.token_uri":           "https://oauth2.googleapis.com/token",
				"json_data.0.authentication_type": "jwt",
				"json_data.0.default_project":     "default-project",
				"json_data.0.client_email":        "client-email@default-project.iam.gserviceaccount.com",
				"secure_json_data.0.private_key":  "-----BEGIN PRIVATE KEY-----\nprivate-key\n-----END PRIVATE KEY-----\n",
			},
		},
		{
			resource: "grafana_data_source.github",
			config: `
			resource "grafana_data_source" "github" {
				type = "grafana-github-datasource"
				name = "github"
				json_data {
					github_url = "https://test-github.com"
				}
				secure_json_data {
					access_token = "token for github"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                            "grafana-github-datasource",
				"name":                            "github",
				"json_data.0.github_url":          "https://test-github.com",
				"secure_json_data.0.access_token": "token for github",
			},
			additionalChecks: []resource.TestCheckFunc{
				func(s *terraform.State) error {
					githubURL := dataSource.JSONData["githubUrl"].(string)
					if githubURL != "https://test-github.com" {
						return fmt.Errorf("bad github_url: %s. Expected: %s", githubURL, "https://test-github.com")
					}
					return nil
				},
			},
		},
		{
			resource: "grafana_data_source.athena",
			config: `
			resource "grafana_data_source" "athena" {
				type = "grafana-athena-datasource"
				name = "athena"
				json_data {
					default_region            = "us-east-1"
					auth_type                 = "keys"
					assume_role_arn           = "arn:aws:sts::*:assumed-role/*/*"
					external_id               = "abc123"
					catalog                   = "my-catalog"
					workgroup                 = "my-workgroup"
					database                  = "my-database"
					output_location           = "s3://my-bucket"
				}
				secure_json_data {
					access_key = "123"
					secret_key = "456"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                          "grafana-athena-datasource",
				"name":                          "athena",
				"json_data.0.default_region":    "us-east-1",
				"json_data.0.auth_type":         "keys",
				"json_data.0.assume_role_arn":   "arn:aws:sts::*:assumed-role/*/*",
				"json_data.0.external_id":       "abc123",
				"json_data.0.catalog":           "my-catalog",
				"json_data.0.workgroup":         "my-workgroup",
				"json_data.0.database":          "my-database",
				"json_data.0.output_location":   "s3://my-bucket",
				"secure_json_data.0.access_key": "123",
				"secure_json_data.0.secret_key": "456",
			},
		},
		{
			resource: "grafana_data_source.azure",
			config: `
			resource "grafana_data_source" "azure" {
				type = "grafana-azure-monitor-datasource"
				name = "azure"
				json_data {
					client_id = "lorem-ipsum"
					cloud_name = "azuremonitor"
					subscription_id = "lorem-ipsum"
					tenant_id = "lorem-ipsum"
				}
				secure_json_data {
					client_secret = "lorem-ipsum"
				}
			}
			`,
			attrChecks: map[string]string{
				"type":                        "grafana-azure-monitor-datasource",
				"name":                        "azure",
				"json_data.0.client_id":       "lorem-ipsum",
				"json_data.0.cloud_name":      "azuremonitor",
				"json_data.0.subscription_id": "lorem-ipsum",
				"json_data.0.tenant_id":       "lorem-ipsum",
			},
		},
	}

	// Iterate over the provided configurations for datasources
	for _, test := range resourceTests {
		t.Run(test.resource, func(t *testing.T) {
			// Always check that the resource was created and that `id` is a number
			checks := []resource.TestCheckFunc{
				testAccDataSourceCheckExists(test.resource, &dataSource),
				resource.TestMatchResourceAttr(
					test.resource,
					"id",
					idRegexp,
				),
				resource.TestMatchResourceAttr(
					test.resource,
					"uid",
					uidRegexp,
				),
			}

			// Add custom checks for specified attribute values
			for attr, value := range test.attrChecks {
				checks = append(checks, resource.TestCheckResourceAttr(
					test.resource,
					attr,
					value,
				))
			}

			// TODO: Make parallelizable
			resource.Test(t, resource.TestCase{
				ProviderFactories: testAccProviderFactories,
				CheckDestroy:      testAccDataSourceCheckDestroy(&dataSource),
				Steps: []resource.TestStep{
					{
						Config: test.config,
						Check: resource.ComposeAggregateTestCheckFunc(
							append(checks, test.additionalChecks...)...,
						),
					},
					// Test import using ID
					{
						ResourceName:      test.resource,
						ImportState:       true,
						ImportStateVerify: test.verifyImport,
						// Ignore sensitive attributes, we mostly only care about "json_data_encoded"
						ImportStateVerifyIgnore: []string{"secure_json_data_encoded", "http_headers."},
					},
					// Test import using UID
					{
						ResourceName:      test.resource,
						ImportState:       true,
						ImportStateVerify: test.verifyImport,
						// Ignore sensitive attributes, we mostly only care about "json_data_encoded"
						ImportStateVerifyIgnore: []string{"secure_json_data_encoded", "http_headers."},
						ImportStateIdFunc: func(s *terraform.State) (string, error) {
							rs, ok := s.RootModule().Resources[test.resource]
							if !ok {
								return "", fmt.Errorf("resource not found: %s", test.resource)
							}

							if rs.Primary.ID == "" {
								return "", fmt.Errorf("resource id not set")
							}
							return rs.Primary.Attributes["uid"], nil
						},
					},
				},
			})
		})
	}
}

func TestDatasourceMigrationV0(t *testing.T) {
	IsUnitTest(t)

	cases := []struct {
		name     string
		state    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "no json data",
			state: map[string]interface{}{
				"name": "test",
				"type": "prometheus",
			},
			expected: map[string]interface{}{
				"name": "test",
				"type": "prometheus",
			},
		},
		{
			name: "no tsdb fields",
			state: map[string]interface{}{
				"name": "test",
				"type": "prometheus",
				"json_data": []map[string]interface{}{
					{
						"url": "http://localhost:9090",
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test",
				"type": "prometheus",
				"json_data": []map[string]interface{}{
					{
						"url": "http://localhost:9090",
					},
				},
			},
		},
		{
			name: "nil or empty tsdb fields",
			state: map[string]interface{}{
				"name": "test",
				"type": "test",
				"json_data": []map[string]interface{}{
					{
						"tsdb_version":    "",
						"tsdb_resolution": nil,
						"url":             "http://localhost:9090",
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test",
				"type": "test",
				"json_data": []map[string]interface{}{
					{
						"tsdb_version":    0,
						"tsdb_resolution": 0,
						"url":             "http://localhost:9090",
					},
				},
			},
		},
		{
			name: "already int",
			state: map[string]interface{}{
				"name": "test",
				"type": "test",
				"json_data": []map[string]interface{}{
					{
						"tsdb_version":    0,
						"tsdb_resolution": 2,
						"url":             "http://localhost:9090",
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test",
				"type": "test",
				"json_data": []map[string]interface{}{
					{
						"tsdb_version":    0,
						"tsdb_resolution": 2,
						"url":             "http://localhost:9090",
					},
				},
			},
		},
		{
			name: "migration",
			state: map[string]interface{}{
				"name": "test",
				"type": "test",
				"json_data": []map[string]interface{}{
					{
						"tsdb_version":    "0",
						"tsdb_resolution": "2",
						"url":             "http://localhost:9090",
					},
				},
			},
			expected: map[string]interface{}{
				"name": "test",
				"type": "test",
				"json_data": []map[string]interface{}{
					{
						"tsdb_version":    0,
						"tsdb_resolution": 2,
						"url":             "http://localhost:9090",
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := resourceDataSourceV0Upgrader.Upgrade(nil, tc.expected, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("Expected %#v, got %#v", tc.expected, actual)
			}
		})
	}
}

func TestAccDataSource_changeUID(t *testing.T) {
	CheckOSSTestsEnabled(t)

	var dataSource gapi.DataSource

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccDataSourceCheckDestroy(&dataSource),
		Steps: []resource.TestStep{
			{
				Config: `
	resource "grafana_data_source" "test" {
		name = "test-change-uid"
		type = "prometheus"
		url  = "http://localhost:9090"
		uid  = "initial-uid"
	}`,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCheckExists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr("grafana_data_source.test", "uid", "initial-uid"),
				),
			},
			{
				Config: `
	resource "grafana_data_source" "test" {
		name = "test-change-uid"
		type = "prometheus"
		url  = "http://localhost:9090"
		uid  = "changed-uid"
	}`,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceCheckExists("grafana_data_source.test", &dataSource),
					resource.TestCheckResourceAttr("grafana_data_source.test", "uid", "changed-uid"),
				),
			},
		},
	})
}

func testAccDataSourceCheckExists(rn string, dataSource *gapi.DataSource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		gotDataSource, err := client.DataSource(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*dataSource = *gotDataSource

		return nil
	}
}

func testAccDataSourceCheckDestroy(dataSource *gapi.DataSource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
		_, err := client.DataSource(dataSource.ID)
		if err == nil {
			return fmt.Errorf("data source still exists")
		}
		return nil
	}
}