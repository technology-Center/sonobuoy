/*
Copyright the Sonobuoy contributors 2019

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

package results

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/daemonset"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/job"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

	"github.com/kylelemons/godebug/pretty"
)

var update = flag.Bool("update", false, "update golden files")

// TestPostProcessPlugin runs a series of checks against basic combinations
// of options: (job|daemonset)+(raw|junit)+(specify a specific file or not)
// and confirms the resulting Item is accurate.
func TestPostProcessPlugin(t *testing.T) {
	getPlugin := func(key, pluginDriver, format string, outputFiles []string) plugin.Interface {
		switch pluginDriver {
		case "job":
			return &job.Plugin{Base: driver.Base{
				Definition: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						PluginName:   key,
						ResultFormat: format,
						ResultFiles:  outputFiles,
					},
				},
			}}
		case "daemonset":
			return &daemonset.Plugin{Base: driver.Base{
				Definition: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						PluginName:   key,
						ResultFormat: format,
						ResultFiles:  outputFiles,
					},
				},
			}}
		default:
			t.Fatalf("Invalid driver specified: %v", pluginDriver)
		}
		return nil
	}

	mockDataDir := func(key string) string {
		return filepath.Join("testdata", "mockResults")
	}
	expectResults := func(key string) string {
		return filepath.Join("testdata", "mockResults", "plugins", key, key+".golden.json")
	}

	testCases := []struct {
		desc         string
		plugin       plugin.Interface
		expectedErrs []string

		// key is used to lookup both the directory and the expected results.
		key string
	}{
		{
			desc:   "Job junit with 2 files, all processed",
			key:    "job-junit-02",
			plugin: getPlugin("job-junit-02", "job", "junit", []string{}),
		}, {
			desc:   "Job junit with 1 file processed, others ignored",
			key:    "job-junit-01",
			plugin: getPlugin("job-junit-01", "job", "junit", []string{"output.xml"}),
		}, {
			desc:   "Job junit with 2 files processed, others ignored",
			key:    "job-junit-03",
			plugin: getPlugin("job-junit-03", "job", "junit", []string{"output.xml", "output2.xml"}),
		}, {
			desc:   "Daemonset junit with 2 files, all processed",
			key:    "ds-junit-02",
			plugin: getPlugin("ds-junit-02", "daemonset", "junit", []string{}),
		}, {
			desc:   "Daemonset junit with 1 file processed, others ignored",
			key:    "ds-junit-01",
			plugin: getPlugin("ds-junit-01", "daemonset", "junit", []string{"output.xml"}),
		}, {
			desc:   "Daemonset junit with 2 files processed, others ignored",
			key:    "ds-junit-03",
			plugin: getPlugin("ds-junit-03", "daemonset", "junit", []string{"output.xml", "output2.xml"}),
		}, {
			desc:   "Job raw with 2 files, all processed",
			key:    "job-raw-02",
			plugin: getPlugin("job-raw-02", "job", "raw", []string{}),
		}, {
			desc:   "Job raw with 1 file processed, others ignored",
			key:    "job-raw-01",
			plugin: getPlugin("job-raw-01", "job", "raw", []string{"output.xml"}),
		}, {
			desc:   "Job raw with 2 files processed, others ignored",
			key:    "job-raw-03",
			plugin: getPlugin("job-raw-03", "job", "raw", []string{"output.xml", "output2.xml"}),
		}, {
			desc:   "Job default with 2 files, all processed",
			key:    "job-default-02",
			plugin: getPlugin("job-default-02", "job", "", []string{}),
		}, {
			desc:   "Job default with 1 file processed, others ignored",
			key:    "job-default-01",
			plugin: getPlugin("job-default-01", "job", "", []string{"output.xml"}),
		}, {
			desc:   "Daemonset raw with 2 files, all processed",
			key:    "ds-raw-02",
			plugin: getPlugin("ds-raw-02", "daemonset", "raw", []string{}),
		}, {
			desc:   "Daemonset raw with 1 file processed, others ignored",
			key:    "ds-raw-01",
			plugin: getPlugin("ds-raw-01", "daemonset", "raw", []string{"output.xml"}),
		}, {
			desc:   "Daemonset raw with 2 files processed, others ignored",
			key:    "ds-raw-03",
			plugin: getPlugin("ds-raw-03", "daemonset", "raw", []string{"output.xml", "output2.xml"}),
		}, {
			desc:   "Job has errors dir considered",
			key:    "job-errors",
			plugin: getPlugin("job-errors", "job", "junit", []string{}),
		}, {
			desc:   "DS has errors dir considered, still processes results for other nodes",
			key:    "ds-errors-01",
			plugin: getPlugin("ds-errors-01", "daemonset", "junit", []string{}),
		}, {
			desc:   "DS has errors dir considered every each node",
			key:    "ds-errors-02",
			plugin: getPlugin("ds-errors-02", "daemonset", "junit", []string{}),
		}, {
			desc:   "Timeout errors cause timeout status",
			key:    "job-timeout",
			plugin: getPlugin("job-timeout", "job", "junit", []string{}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			item, errs := PostProcessPlugin(tc.plugin, mockDataDir(tc.key))
			if len(errs) > 0 {
				for _, e := range errs {
					t.Errorf("Unexpected error: %v", e)
				}
				t.FailNow()
			}
			if *update {
				// Update all the golden files instead of actually testing against them.
				itemBytes, err := json.MarshalIndent(item, "", "")
				if err != nil {
					t.Fatalf("Failed to marshal item: %v", err)
				}
				ioutil.WriteFile(expectResults(tc.key), itemBytes, 0666)
			} else {
				// Read in golden file and unmarshal. Easier to debug differences in the items than
				// comparing the bytes directly.
				fileData, err := ioutil.ReadFile(expectResults(tc.key))
				if err != nil {
					t.Fatalf("Failed to read golden file %v: %v", expectResults(tc.key), err)
				}
				var expectedItem Item
				err = json.Unmarshal(fileData, &expectedItem)
				if err != nil {
					t.Fatalf("Failed to unmarshal golden file %v: %v", expectResults(tc.key), err)
				}
				if diff := pretty.Compare(expectedItem, item); diff != "" {
					t.Fatalf("\n\n%s\n", diff)
				}
			}
		})
	}
}

func TestAggregateStatus(t *testing.T) {
	tcs := []struct {
		desc     string
		input    []Item
		expected string

		// Ensures the items are actually updated despite their initial values.
		expectedItems []Item
	}{
		{
			desc:     "Empty defaults to unknown",
			expected: StatusUnknown,
		}, {
			desc:          "Single pass passes",
			input:         []Item{{Status: StatusPassed}},
			expectedItems: []Item{{Status: StatusPassed}},
			expected:      StatusPassed,
		}, {
			desc:          "Single fail fails",
			input:         []Item{{Status: StatusFailed}},
			expectedItems: []Item{{Status: StatusFailed}},
			expected:      StatusFailed,
		}, {
			desc:          "Single unknown is unknown",
			input:         []Item{{Status: StatusUnknown}},
			expectedItems: []Item{{Status: StatusUnknown}},
			expected:      StatusUnknown,
		}, {
			desc:          "Misc other values pass",
			input:         []Item{{Status: "foobar"}},
			expectedItems: []Item{{Status: "foobar"}},
			expected:      StatusPassed,
		}, {
			desc:          "Timeout bubbles up as failure",
			input:         []Item{{Status: "timeout"}},
			expectedItems: []Item{{Status: "timeout"}},
			expected:      StatusFailed,
		}, {
			desc: "Single failure in group causes failure",
			input: []Item{
				{Status: StatusPassed},
				{Status: StatusFailed},
			},
			expectedItems: []Item{
				{Status: StatusPassed},
				{Status: StatusFailed},
			},
			expected: StatusFailed,
		}, {
			desc: "Single unknown in group causes unknown",
			input: []Item{
				{Status: StatusPassed},
				{Status: StatusUnknown},
			},
			expectedItems: []Item{
				{Status: StatusPassed},
				{Status: StatusUnknown},
			},
			expected: StatusUnknown,
		}, {
			desc: "Failure takes priority over unknown",
			input: []Item{
				{Status: StatusPassed},
				{Status: StatusUnknown},
				{Status: StatusFailed},
			},
			expectedItems: []Item{
				{Status: StatusPassed},
				{Status: StatusUnknown},
				{Status: StatusFailed},
			},
			expected: StatusFailed,
		}, {
			desc: "Nested failure causes failure",
			input: []Item{
				{
					Status: StatusPassed,
					Items: []Item{
						{Status: StatusFailed},
					},
				},
				{Status: StatusPassed},
			},
			expectedItems: []Item{
				{
					Status: StatusFailed,
					Items: []Item{
						{Status: StatusFailed},
					},
				},
				{Status: StatusPassed},
			},
			expected: StatusFailed,
		}, {
			desc: "Nested unknown causes unknown",
			input: []Item{
				{
					Status: StatusPassed,
					Items: []Item{
						{Status: StatusUnknown},
					},
				},
				{Status: StatusPassed},
			},
			expectedItems: []Item{
				{
					Status: StatusUnknown,
					Items: []Item{
						{Status: StatusUnknown},
					},
				},
				{Status: StatusPassed},
			},
			expected: StatusUnknown,
		}, {
			desc: "Deep branches should aggregate their items and return if failure",
			input: []Item{
				{
					Name:   "top of a branch",
					Status: StatusPassed,
					Items: []Item{
						{
							Name: "passing node",
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								},
							},
						},
						{
							Name: "failing node",
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								}, {
									Name:   "second leaf fails and should fail branch",
									Status: StatusFailed,
								}, {
									Name:   "third leaf passes as well",
									Status: StatusPassed,
								},
							},
						},
					},
				},
			},
			expectedItems: []Item{
				{
					Name:   "top of a branch",
					Status: StatusFailed,
					Items: []Item{
						{
							Name:   "passing node",
							Status: StatusPassed,
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								},
							},
						},
						{
							Name:   "failing node",
							Status: StatusFailed,
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								}, {
									Name:   "second leaf fails and should fail branch",
									Status: StatusFailed,
								}, {
									Name:   "third leaf passes as well",
									Status: StatusPassed,
								},
							},
						},
					},
				},
			},
			expected: StatusFailed,
		}, {
			desc: "Deep branches should aggregate their items and return if unknown",
			input: []Item{
				{
					Name:   "top of a branch",
					Status: StatusPassed,
					Items: []Item{
						{
							Name: "passing node",
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								},
							},
						},
						{
							Name: "unknown node",
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								}, {
									Name:   "second leaf unknown and should cause branch to be unknown",
									Status: StatusUnknown,
								}, {
									Name:   "third leaf passes as well",
									Status: StatusPassed,
								},
							},
						},
					},
				},
			},
			expectedItems: []Item{
				{
					Name:   "top of a branch",
					Status: StatusUnknown,
					Items: []Item{
						{
							Name:   "passing node",
							Status: StatusPassed,
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								},
							},
						},
						{
							Name:   "unknown node",
							Status: StatusUnknown,
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								}, {
									Name:   "second leaf unknown and should cause branch to be unknown",
									Status: StatusUnknown,
								}, {
									Name:   "third leaf passes as well",
									Status: StatusPassed,
								},
							},
						},
					},
				},
			},
			expected: StatusUnknown,
		}, {
			desc: "Leaf nodes with empty status get changed to unknown",
			input: []Item{
				{
					Name: "unknown node",
					Items: []Item{
						{
							Name: "first leaf no status",
						},
					},
				},
			},
			expectedItems: []Item{
				{
					Name:   "unknown node",
					Status: StatusUnknown,
					Items: []Item{
						{
							Name:   "first leaf no status",
							Status: StatusUnknown,
						},
					},
				},
			},
			expected: StatusUnknown,
		}, {
			desc: "Processes all nodes even after seeing first failure",
			input: []Item{
				{
					Name: "DS plugin",
					Items: []Item{
						{
							Name: "node1",
							Items: []Item{
								{Name: "foo", Status: StatusFailed},
							},
						},
						{
							Name: "node2",
							Items: []Item{
								{Name: "foo", Status: StatusFailed},
							},
						},
					},
				},
			},
			expectedItems: []Item{
				{
					Name:   "DS plugin",
					Status: StatusFailed,
					Items: []Item{
						{
							Name:   "node1",
							Status: StatusFailed,
							Items: []Item{
								{Name: "foo", Status: StatusFailed},
							},
						},
						{
							Name:   "node2",
							Status: StatusFailed,
							Items: []Item{
								{Name: "foo", Status: StatusFailed},
							},
						},
					},
				},
			},
			expected: StatusFailed,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			out := aggregateStatus(tc.input...)
			if out != tc.expected {
				t.Errorf("Expected %v but got %v", tc.expected, out)
			}

			if diff := pretty.Compare(tc.expectedItems, tc.input); diff != "" {
				t.Errorf("\n\n%s\n", diff)
			}
		})
	}
}
