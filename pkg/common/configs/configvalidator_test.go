/*
 Licensed to the Apache Software Foundation (ASF) under one
 or more contributor license agreements.  See the NOTICE file
 distributed with this work for additional information
 regarding copyright ownership.  The ASF licenses this file
 to you under the Apache License, Version 2.0 (the
 "License"); you may not use this file except in compliance
 with the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package configs

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/apache/yunikorn-core/pkg/common"
	"github.com/apache/yunikorn-core/pkg/common/resources"
)

func TestCheckResourceConfigurationsForQueue(t *testing.T) {
	negativeResourceMap := map[string]string{"memory": "-50", "vcores": "33"}
	resourceMapWithSyntaxError := map[string]string{"memory": "ten", "vcores": ""}
	higherResourceMap := map[string]string{"memory": "50", "vcores": "33"}
	undefinedVcoresResourceMap := map[string]string{"memory": "20"}
	lowerResourceMap := map[string]string{"memory": "10", "vcores": "30"}
	testCases := []struct {
		name          string
		current       QueueConfig
		errorExpected bool
	}{
		{"Negative guaranteed resource", QueueConfig{
			Resources: Resources{
				Guaranteed: negativeResourceMap,
			},
		}, true},
		{"Negative max resource", QueueConfig{
			Resources: Resources{
				Max: negativeResourceMap,
			},
		}, true},
		{"Nil guaranteed resource", QueueConfig{
			Resources: Resources{
				Max: lowerResourceMap,
			},
		}, false},
		{"Nil max resource", QueueConfig{
			Resources: Resources{
				Guaranteed: lowerResourceMap,
			},
		}, false},
		{"Syntax error in guaranteed resource", QueueConfig{
			Resources: Resources{
				Guaranteed: resourceMapWithSyntaxError,
			},
		}, true},
		{"Syntax error in max resource", QueueConfig{
			Resources: Resources{
				Max: resourceMapWithSyntaxError,
			},
		}, true},
		{"Higher guaranteed resource in child queues", QueueConfig{
			Resources: Resources{
				Guaranteed: lowerResourceMap,
			},
			Queues: []QueueConfig{{
				Resources: Resources{
					Guaranteed: higherResourceMap,
				},
			}},
		}, true},
		{"Higher sum of guaranteed resource in child queues than the parent's guaranteed", QueueConfig{
			Resources: Resources{
				Max:        higherResourceMap,
				Guaranteed: lowerResourceMap,
			},
			Queues: []QueueConfig{{
				Resources: Resources{
					Max:        lowerResourceMap,
					Guaranteed: lowerResourceMap,
				},
			}, {
				Resources: Resources{
					Max:        lowerResourceMap,
					Guaranteed: lowerResourceMap,
				},
			}},
		}, true},
		{"Higher sum of guaranteed resource in child queues than the parent's Max when guaranteed is undefined", QueueConfig{
			Resources: Resources{
				Max:        higherResourceMap,
				Guaranteed: undefinedVcoresResourceMap,
			},
			Queues: []QueueConfig{{
				Resources: Resources{
					Max:        lowerResourceMap,
					Guaranteed: lowerResourceMap,
				},
			}, {
				Resources: Resources{
					Max:        lowerResourceMap,
					Guaranteed: lowerResourceMap,
				},
			}},
		}, true},
		{"Higher max resource in child queues", QueueConfig{
			Resources: Resources{
				Max: lowerResourceMap,
			},
			Queues: []QueueConfig{{
				Resources: Resources{
					Max: higherResourceMap,
				},
			}},
		}, true},
		{"Higher guaranteed than max resource", QueueConfig{
			Resources: Resources{
				Max:        lowerResourceMap,
				Guaranteed: higherResourceMap,
			},
		}, true},
		{"Valid configuration",
			QueueConfig{
				Resources: Resources{
					Max:        higherResourceMap,
					Guaranteed: lowerResourceMap,
				},
			},
			false},
		{"One level skipped while setting max resource",
			createQueueWithSkippedMaxRes(),
			true},
		{"Sum of child guaranteed higher than parent max",
			createQueueWithSumGuaranteedHigherThanParentMax(),
			true},
		{"One level skipped while setting guaranteed resource",
			createQueueWithSkippedGuaranteedRes(),
			true},
		{"Parent with undefined guaranteed resources",
			createQueueWithUndefinedGuaranteedResources(),
			false},
		{"Nil guaranteed resources for both parent and child",
			QueueConfig{
				Resources: Resources{
					Guaranteed: nil,
					Max:        higherResourceMap,
				},
				Queues: []QueueConfig{
					{
						Resources: Resources{
							Guaranteed: nil,
							Max:        lowerResourceMap,
						},
					},
				},
			},
			false,
		},
		{"Empty guaranteed resources for both parent and child",
			QueueConfig{
				Resources: Resources{
					Guaranteed: map[string]string{},
					Max:        higherResourceMap,
				},
				Queues: []QueueConfig{
					{
						Resources: Resources{
							Guaranteed: map[string]string{},
							Max:        lowerResourceMap,
						},
					},
				},
			},
			false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := checkQueueResource(tc.current, nil)
			if tc.errorExpected {
				assert.Assert(t, err != nil, "An error is expected")
			} else {
				assert.NilError(t, err, "No error is expected")
			}
		})
	}
}

func TestCheckQueueMaxApplicationsForQueue(t *testing.T) {
	testCases := []struct {
		name          string
		current       QueueConfig
		errorExpected bool
	}{
		{"Parent maxRunningApps must be larger than child maxRunningApps",
			createQueueWithMaxApplication([4]uint64{1, 2, 3, 4}),
			true},
		{"Valid maxApplication settings: Parent maxRunningApps larger than child maxRunningApps",
			createQueueWithMaxApplication([4]uint64{4, 3, 2, 1}),
			false},
		{"Valid maxApplication settings: Parent maxRunningApps can be 0",
			createQueueWithMaxApplication([4]uint64{0, 3, 2, 1}),
			false},
		{"InValid maxApplication settings: child maxRunningApps cannot be 0",
			createQueueWithMaxApplication([4]uint64{4, 3, 2, 0}),
			true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkQueueMaxApplications(tc.current)
			if tc.errorExpected {
				assert.Assert(t, err != nil, "An error is expected")
			} else {
				assert.NilError(t, err, "No error is expected")
			}
		})
	}
}

func TestGetLongestPlacementPath(t *testing.T) {
	staticPaths, err := getLongestPlacementPaths(createPlacementRules())
	assert.NilError(t, err)
	assert.Equal(t, 2, len(staticPaths))

	path0 := staticPaths[0]
	assert.Equal(t, "root.users", path0.path)
	assert.Equal(t, 0, path0.ruleNo)
	assert.Equal(t, "fixed->tag->user", path0.ruleChain)
	assert.Equal(t, false, path0.create)
	path1 := staticPaths[1]
	assert.Equal(t, "root.admins.dev", path1.path)
	assert.Equal(t, 1, path1.ruleNo)
	assert.Equal(t, "fixed->fixed", path1.ruleChain)
	assert.Equal(t, true, path1.create)

	// illegal: two "fixed" with fully qualified path
	illegal := []PlacementRule{
		{
			Name:   "fixed",
			Value:  "root.dev",
			Create: true,
			Parent: &PlacementRule{
				Name:  "fixed",
				Value: "root.admins",
			},
		},
	}
	_, err = getLongestPlacementPaths(illegal)
	assert.ErrorContains(t, err, "illegal fully qualified 'fixed' rule")
}

func TestCheckQueueHierarchyForPlacement(t *testing.T) {
	queues := createQueueConfig()

	// case #1 - referring to existing queue which is a parent
	parts := strings.Split("root.users", DOT)
	result, queueName := checkQueueHierarchyForPlacement(parts, false, false, queues, nil)
	assert.Equal(t, errQueueNotLeaf, result)
	assert.Equal(t, "", queueName)

	// case #2 - referring to an existing queue which is a leaf
	parts = strings.Split("root.default", DOT)
	result, queueName = checkQueueHierarchyForPlacement(parts, true, false, queues, nil)
	assert.Equal(t, placementOK, result)
	assert.Equal(t, "", queueName)

	// case #3 - referring a path which is incomplete in the hierarchy, "users" is parent, create = true
	parts = strings.Split("root.users.alice", DOT)
	result, queueName = checkQueueHierarchyForPlacement(parts, true, false, queues, nil)
	assert.Equal(t, placementOK, result)
	assert.Equal(t, "", queueName)

	// case #4 - referring a path which is incomplete in the hierarchy, "users" is parent, create = false
	result, queueName = checkQueueHierarchyForPlacement(parts, false, false, queues, nil)
	assert.Equal(t, errNonExistingQueue, result)
	assert.Equal(t, "", queueName)

	// case #5 - referring a path which is incomplete in the hierarchy, "users" is leaf, create = true
	queues[0].Queues[0].Parent = false
	result, queueName = checkQueueHierarchyForPlacement(parts, true, false, queues, nil)
	assert.Equal(t, errLastQueueLeaf, result)
	assert.Equal(t, "users", queueName)

	// case #6 - referring a path which is incomplete in the hierarchy, "users" is leaf, create = false
	result, queueName = checkQueueHierarchyForPlacement(parts, false, false, queues, nil)
	assert.Equal(t, errLastQueueLeaf, result)
	assert.Equal(t, "users", queueName)

	// case #7 - hierarchy is long enough, but no matching queue found, create = true
	parts = strings.Split("root.devs.test", DOT)
	result, queueName = checkQueueHierarchyForPlacement(parts, true, false, queues, nil)
	assert.Equal(t, placementOK, result)
	assert.Equal(t, "", queueName)

	// case #8 - hierarchy is long enough, but no matching queue found, create = false
	result, queueName = checkQueueHierarchyForPlacement(parts, false, false, queues, nil)
	assert.Equal(t, errNonExistingQueue, result)
	assert.Equal(t, "", queueName)

	// case #9 - rule chain ends with a dynamic part, last queue is a leaf
	parts = strings.Split("root.users", DOT)
	result, queueName = checkQueueHierarchyForPlacement(parts, false, true, queues, nil)
	assert.Equal(t, errQueueNotLeaf, result)
	assert.Equal(t, "", queueName)

	// case #10 - rule chain ends with a dynamic part, last queue is a parent
	queues[0].Queues[0].Parent = true
	parts = strings.Split("root.users", DOT)
	result, queueName = checkQueueHierarchyForPlacement(parts, false, true, queues, nil)
	assert.Equal(t, placementOK, result)
	assert.Equal(t, "", queueName)
}

func TestCheckPlacementRules(t *testing.T) {
	conf := &PartitionConfig{
		PlacementRules: createPlacementRules(),
		Queues:         createQueueConfig(),
	}

	// default case, no error
	err := checkPlacementRules(conf)
	assert.NilError(t, err)

	// referencing "root.users", but "users" is a leaf
	conf.PlacementRules = []PlacementRule{
		{
			Name:  "fixed",
			Value: "root.users",
		},
	}
	err = checkPlacementRules(conf)
	assert.ErrorContains(t, err, "placement rule no. #0 (fixed) references a queue (root.users) which is not a leaf")

	// referencing "root.admins.dev" which doesn't exist but 'create' is false
	conf.PlacementRules = createPlacementRules()
	conf.PlacementRules[1].Create = false
	err = checkPlacementRules(conf)
	assert.ErrorContains(t, err, "placement rule no. #1 (fixed->fixed) references non-existing queues (root.admins.dev) and create is 'false'")

	// referencing "root.default", but queues under "default" cannot be created due to "default" being leaf
	conf.PlacementRules = []PlacementRule{
		{
			Name:  "fixed",
			Value: "root.default.leaf",
		},
	}
	err = checkPlacementRules(conf)
	assert.ErrorContains(t, err, "placement rule no. #0 (fixed) references non-existing queues (root.default.leaf) which cannot be created because the last queue (default) in the hierarchy is a leaf")

	// two "fixed" rule in a chain with both having fully qualified queues
	conf.PlacementRules = []PlacementRule{
		{
			Name:  "fixed",
			Value: "root.default.leaf",
			Parent: &PlacementRule{
				Name:  "fixed",
				Value: "root.default",
			},
		},
	}
	err = checkPlacementRules(conf)
	assert.ErrorContains(t, err, "illegal fully qualified 'fixed' rule with value root.default.leaf")
}

func createQueueConfig() []QueueConfig {
	return []QueueConfig{
		{
			Name:   "root",
			Parent: true,
			Queues: []QueueConfig{
				{
					Name:   "users",
					Parent: true,
				},
				{
					Name:   "devs",
					Parent: true,
					Queues: []QueueConfig{
						{
							Name:   "yunikorn",
							Parent: true,
						},
					},
				},
				{
					Name:   "default",
					Parent: false,
				},
				{
					Name:   "admins",
					Parent: true,
				},
			},
		},
	}
}

func createPlacementRules() []PlacementRule {
	return []PlacementRule{
		{
			Name: "user",
			Parent: &PlacementRule{
				Name:  "tag",
				Value: "namespace",
				Parent: &PlacementRule{
					Name:  "fixed",
					Value: "root.users",
				},
			},
		},
		{
			Name:   "fixed",
			Value:  "dev",
			Create: true,
			Parent: &PlacementRule{
				Name:  "fixed",
				Value: "admins",
			},
		},
	}
}

func createQueueWithSkippedMaxRes() QueueConfig {
	child1MaxMap := map[string]string{"memory": "150"}
	parentMaxMap := map[string]string{"memory": "100"}
	child1 := QueueConfig{
		Resources: Resources{
			Max: child1MaxMap,
		},
		Name: "child1",
	}
	parent1 := QueueConfig{
		Queues: []QueueConfig{child1},
		Name:   "parent1",
	}
	parent := QueueConfig{
		Resources: Resources{
			Max: parentMaxMap,
		},
		Queues: []QueueConfig{parent1},
		Name:   "parent",
	}
	root := QueueConfig{
		Queues: []QueueConfig{parent},
		Name:   RootQueue,
	}
	return root
}

func createQueueWithSumGuaranteedHigherThanParentMax() QueueConfig {
	child1GuaranteedMap := map[string]string{"memory": "50"}
	child2GuaranteedMap := map[string]string{"memory": "40"}
	parentMaxMap := map[string]string{"memory": "100"}
	parent1MaxMap := map[string]string{"memory": "80"}
	child1 := QueueConfig{
		Resources: Resources{
			Guaranteed: child1GuaranteedMap,
		},
		Name: "child1",
	}
	child2 := QueueConfig{
		Resources: Resources{
			Guaranteed: child2GuaranteedMap,
		},
		Name: "child1",
	}
	parent1 := QueueConfig{
		Queues: []QueueConfig{child1, child2},
		Name:   "parent1",
		Resources: Resources{
			Max: parent1MaxMap,
		},
	}
	parent := QueueConfig{
		Resources: Resources{
			Max: parentMaxMap,
		},
		Queues: []QueueConfig{parent1},
		Name:   "parent",
	}
	root := QueueConfig{
		Queues: []QueueConfig{parent},
		Name:   RootQueue,
	}
	return root
}

func createQueueWithSkippedGuaranteedRes() QueueConfig {
	child1MaxMap := map[string]string{"memory": "150"}
	parentMaxMap := map[string]string{"memory": "100"}
	child1 := QueueConfig{
		Resources: Resources{
			Guaranteed: child1MaxMap,
		},
		Name: "child1",
	}
	parent1 := QueueConfig{
		Queues: []QueueConfig{child1},
		Name:   "parent1",
	}
	parent := QueueConfig{
		Resources: Resources{
			Guaranteed: parentMaxMap,
		},
		Queues: []QueueConfig{parent1},
		Name:   "parent",
	}
	root := QueueConfig{
		Queues: []QueueConfig{parent},
		Name:   RootQueue,
	}
	return root
}

func createQueueWithUndefinedGuaranteedResources() QueueConfig {
	parentMax := map[string]string{"memory": "100", "vcores": "100"}
	parentGuaranteed := map[string]string{"memory": "100"}
	childMax := map[string]string{"memory": "80", "vcores": "80"}
	childGuaranteed := map[string]string{"memory": "50", "vcores": "50"}
	child1 := QueueConfig{
		Resources: Resources{
			Guaranteed: childGuaranteed,
			Max:        childMax,
		},
		Name: "child1",
	}
	child2 := QueueConfig{
		Resources: Resources{
			Guaranteed: childGuaranteed,
			Max:        childMax,
		},
		Name: "child2",
	}
	parent := QueueConfig{
		Resources: Resources{
			Guaranteed: parentGuaranteed,
			Max:        parentMax,
		},
		Name:   "parent",
		Queues: []QueueConfig{child1, child2},
	}
	root := QueueConfig{
		Queues: []QueueConfig{parent},
		Name:   RootQueue,
	}
	return root
}
func createQueueWithMaxApplication(maxApplication [4]uint64) QueueConfig {
	child1MaxMap := map[string]string{"memory": "50"}
	parentMaxMap := map[string]string{"memory": "100"}
	child1 := QueueConfig{
		Resources: Resources{
			Guaranteed: child1MaxMap,
		},
		Name:            "child1",
		MaxApplications: maxApplication[3],
	}
	parent1 := QueueConfig{
		Queues:          []QueueConfig{child1},
		Name:            "parent1",
		MaxApplications: maxApplication[2],
	}
	parent := QueueConfig{
		Resources: Resources{
			Guaranteed: parentMaxMap,
		},
		Queues:          []QueueConfig{parent1},
		Name:            "parent",
		MaxApplications: maxApplication[1],
	}
	root := QueueConfig{
		Queues:          []QueueConfig{parent},
		Name:            RootQueue,
		MaxApplications: maxApplication[0],
	}
	return root
}

func TestUserName(t *testing.T) {
	allowedUserNames := []string{
		"username-allowed_99",
		"username",
		"username*regexp",
		"user_name",
		"user@name@",
		"username$$",
	}
	for _, allowed := range allowedUserNames {
		t.Run(allowed, func(t *testing.T) {
			filter := Filter{
				Users:  []string{allowed},
				Groups: []string{"ok"},
			}
			assert.NilError(t, checkPlacementFilter(filter))
		})
	}

	rejectedUserNames := []string{
		"username rejected",
		"",
		"rejected#",
		"rejected!name",
		"!rejected",
		" rejected ",
	}
	for _, rejected := range rejectedUserNames {
		t.Run(rejected, func(t *testing.T) {
			filter := Filter{
				Users:  []string{rejected},
				Groups: []string{"ok"},
			}
			assert.ErrorContains(t, checkPlacementFilter(filter), "invalid rule filter user list")
		})
	}
}

func TestGroupName(t *testing.T) {
	allowedGroupNames := []string{
		"groupname-allowed_99",
		"groupname",
		"groupname*regexp",
		"group_name",
		"group-name",
	}
	for _, allowed := range allowedGroupNames {
		t.Run(allowed, func(t *testing.T) {
			filter := Filter{
				Users:  []string{"ok"},
				Groups: []string{allowed},
			}
			assert.NilError(t, checkPlacementFilter(filter))
		})
	}

	rejectedGroupNames := []string{
		"groupname ",
		"group@name",
		"group name",
		" groupname ",
		"!groupname",
	}
	for _, rejected := range rejectedGroupNames {
		t.Run(rejected, func(t *testing.T) {
			filter := Filter{
				Users:  []string{"ok"},
				Groups: []string{rejected},
			}
			assert.ErrorContains(t, checkPlacementFilter(filter), "invalid rule filter group list")
		})
	}
}

func TestServiceAccountUserName(t *testing.T) {
	allowedUserNames := []string{
		"system:serviceaccounts:username:username-77",
		"system:serviceaccounts:username:regexp*",
		"system:serviceaccounts:username.12:username-77-11",
		"system:serviceaccounts:username.12:username-77-11",
		"system:serviceaccounts:username:username",
	}
	for _, allowed := range allowedUserNames {
		t.Run(allowed, func(t *testing.T) {
			filter := Filter{
				Users:  []string{allowed},
				Groups: []string{"ok"},
			}

			assert.NilError(t, checkPlacementFilter(filter))
		})
	}

	rejectedUserNames := []string{
		"system:serviceaccounts:username!:username",
		"system:serviceaccounts:username xyz:username",
		"system:serviceaccounts: :username",
		"system:serviceaccounts: username:username",
		"system:\\:username",
		" system:serviceaccounts:username:username",
	}
	for _, rejected := range rejectedUserNames {
		t.Run(rejected, func(t *testing.T) {
			filter := Filter{
				Users:  []string{rejected},
				Groups: []string{"ok"},
			}

			assert.ErrorContains(t, checkPlacementFilter(filter), "invalid rule filter user list")
		})
	}
}

func TestServiceAccountGroupName(t *testing.T) {
	allowedGroupNames := []string{
		"system:authenticated",
		"system:unauthenticated",
		"system:serviceaccounts:groupname",
		"system:serviceaccounts:groupname.12",
		"system:serviceaccounts:groupname.12",
		"system:serviceaccounts:groupname-1-test.test",
	}
	for _, allowed := range allowedGroupNames {
		t.Run(allowed, func(t *testing.T) {
			filter := Filter{
				Users:  []string{"ok"},
				Groups: []string{allowed},
			}

			assert.NilError(t, checkPlacementFilter(filter))
		})
	}

	rejectedGroupNames := []string{
		"system:\\:groupname",
		"system:serviceaccounts: groupname",
		"system:!:groupname",
		"system:&:groupname",
		" system:authenticated",
	}
	for _, rejected := range rejectedGroupNames {
		t.Run(rejected, func(t *testing.T) {
			filter := Filter{
				Users:  []string{"ok"},
				Groups: []string{rejected},
			}

			assert.ErrorContains(t, checkPlacementFilter(filter), "invalid rule filter group list")
		})
	}
}

func TestCheckLimitResource(t *testing.T) { //nolint:funlen
	testCases := []struct {
		name     string
		config   QueueConfig
		hasError bool
	}{
		{
			name: "leaf queue user group maxresources are within immediate parent queue user group maxresources",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "100"}},
					map[string]map[string]string{"test-group": {"memory": "100"}}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "50"}},
							map[string]map[string]string{"test-group": {"memory": "50"}}),
						Queues: []QueueConfig{
							{
								Name: "child2",
								Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "10"}},
									map[string]map[string]string{"test-group": {"memory": "10"}}),
							},
						},
					},
				},
			},
			hasError: false,
		},
		{
			name: "leaf queue user group maxresources are within ancestor parent queue user group maxresources",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "100"}},
					map[string]map[string]string{"test-group": {"memory": "100"}}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Queues: []QueueConfig{
							{
								Name: "childA",
								Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "50"}},
									map[string]map[string]string{"test-group": {"memory": "50"}}),
							},
							{
								Name: "childB",
								Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "10"}},
									map[string]map[string]string{"test-group": {"memory": "10"}}),
							},
						},
					},
				},
			},
			hasError: false,
		},
		{
			name: "leaf queue user maxresources exceed parent queue user maxresources",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "100"}},
					map[string]map[string]string{"test-group": {"memory": "100"}}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "200"}},
							map[string]map[string]string{"test-group": {"memory": "50"}}),
					},
				},
			},
			hasError: true,
		},
		{
			name: "leaf queue group maxresources exceed parent queue group maxresources",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "100"}},
					map[string]map[string]string{"test-group": {"memory": "100"}}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "50"}},
							map[string]map[string]string{"test-group": {"memory": "200"}}),
					},
				},
			},
			hasError: true,
		},
		{
			name: "queues at same level maxresources can be greater or less than or equal to the other but with in immediate parent",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "100"}},
					map[string]map[string]string{"test-group": {"memory": "100"}}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "50"}},
							map[string]map[string]string{"test-group": {"memory": "50"}}),
					},
					{
						Name: "child2",
						Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "60"}},
							map[string]map[string]string{"test-group": {"memory": "60"}}),
					},
				},
			},
			hasError: false,
		},
		{
			name: "queues at same level maxresources can be greater or less than or equal to the other but with in ancestor parent",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "100"}},
					map[string]map[string]string{"test-group": {"memory": "100"}}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Queues: []QueueConfig{
							{
								Name: "childA",
								Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "50"}},
									map[string]map[string]string{"test-group": {"memory": "50"}}),
							},
							{
								Name: "childB",
								Limits: createLimitMaxResources(map[string]map[string]string{"test-user": {"memory": "60"}},
									map[string]map[string]string{"test-group": {"memory": "60"}}),
							},
						},
					},
				},
			},
			hasError: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := checkLimitResource(testCase.config, make(map[string]map[string]*resources.Resource), make(map[string]map[string]*resources.Resource), common.Empty)
			if testCase.hasError {
				assert.ErrorContains(t, err, "is greater than immediate or ancestor parent maximum resource")
			} else {
				assert.NilError(t, err)
			}
		})
	}
}

func createLimitMaxResources(users map[string]map[string]string, groups map[string]map[string]string) []Limit {
	var limits []Limit
	for user, limit := range users {
		var users []string
		users = append(users, user)
		limit := Limit{Limit: "user-limit", Users: users, MaxResources: limit}
		limits = append(limits, limit)
	}
	for group, limit := range groups {
		var groups []string
		groups = append(groups, group)
		limit := Limit{Limit: "group-limit", Groups: groups, MaxResources: limit}
		limits = append(limits, limit)
	}
	return limits
}

func createLimitMaxApplications(users map[string]uint64, groups map[string]uint64) []Limit {
	var limits []Limit
	for user, limit := range users {
		var users []string
		users = append(users, user)
		limit := Limit{Limit: "user-limit", Users: users, MaxApplications: limit}
		limits = append(limits, limit)
	}

	for group, limit := range groups {
		var groups []string
		groups = append(groups, group)
		limit := Limit{Limit: "group-limit", Groups: groups, MaxApplications: limit}
		limits = append(limits, limit)
	}
	return limits
}

func TestCheckLimitMaxApplications(t *testing.T) { //nolint:funlen
	testCases := []struct {
		name     string
		config   QueueConfig
		hasError bool
	}{
		{
			name: "leaf queue user group maxapplications are within parent queue user group maxapplications",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxApplications(map[string]uint64{"test-user": 100},
					map[string]uint64{"test-group": 100}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Limits: createLimitMaxApplications(map[string]uint64{"test-user": 100},
							map[string]uint64{"test-group": 100}),
						Queues: []QueueConfig{
							{
								Name: "child2",
								Limits: createLimitMaxApplications(map[string]uint64{"test-user": 100},
									map[string]uint64{"test-group": 100}),
							},
						},
					},
				},
			},
			hasError: false,
		},
		{
			name: "leaf queue user maxapplications exceed parent queue user maxapplications",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxApplications(map[string]uint64{"test-user": 100},
					map[string]uint64{"test-group": 100}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Limits: createLimitMaxApplications(map[string]uint64{"test-user": 200},
							map[string]uint64{"test-group": 50}),
					},
				},
			},
			hasError: true,
		},
		{
			name: "leaf queue group maxapplications exceed parent queue group maxapplications",
			config: QueueConfig{
				Name: "parent",
				Limits: createLimitMaxApplications(map[string]uint64{"test-user": 100},
					map[string]uint64{"test-group": 100}),
				Queues: []QueueConfig{
					{
						Name: "child1",
						Limits: createLimitMaxApplications(map[string]uint64{"test-user": 50},
							map[string]uint64{"test-group": 200}),
					},
				},
			},
			hasError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := checkLimitMaxApplications(testCase.config, make(map[string]map[string]uint64), make(map[string]map[string]uint64), common.Empty)
			if testCase.hasError {
				assert.ErrorContains(t, err, "is greater than immediate or ancestor parent max applications")
			} else {
				assert.NilError(t, err)
			}
		})
	}
}

func TestCheckLimits(t *testing.T) { //nolint:funlen
	testCases := []struct {
		name   string
		config QueueConfig
		errMsg string
	}{
		{
			name: "user group maxresources and maxapplications are within queue limits",
			config: QueueConfig{
				Name: "parent",
				Resources: Resources{
					Max: map[string]string{"memory": "100"},
				},
				MaxApplications: 100,
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user", "*"},
						MaxResources:    map[string]string{"memory": "100"},
						MaxApplications: 100,
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group", "*"},
						MaxResources:    map[string]string{"memory": "100"},
						MaxApplications: 100,
					},
				},
			},
			errMsg: "",
		},
		{
			name: "default queue maxapplications",
			config: QueueConfig{
				Name: "parent",
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 100,
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group"},
						MaxApplications: 100,
					},
				},
			},
			errMsg: "",
		},
		{
			name: "partial fields in maxresources are 0",
			config: QueueConfig{
				Name: "parent",
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 1,
						MaxResources:    map[string]string{"memory": "100", "cpu": "0", "nvidia.com/gpu": "0"},
					},
				},
			},
			errMsg: "",
		},
		{
			name: "all fields in maxresources are 0",
			config: QueueConfig{
				Name: "parent",
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 1,
						MaxResources:    map[string]string{"memory": "0", "cpu": "0"},
					},
				},
			},
			errMsg: "MaxResources is zero",
		},
		{
			name: "both maxresources and maxresources are 0",
			config: QueueConfig{
				Name: "parent",
				Limits: []Limit{
					{
						Limit: "user-limit",
						Users: []string{"test-user"},
					},
				},
			},
			errMsg: "invalid resource combination",
		},
		{
			name: "user maxresources exceed queue limits",
			config: QueueConfig{
				Name: "parent",
				Resources: Resources{
					Max: map[string]string{"memory": "100"},
				},
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 1,
						MaxResources:    map[string]string{"memory": "200"},
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group"},
						MaxApplications: 1,
						MaxResources:    map[string]string{"memory": "100"},
					},
				},
			},
			errMsg: "invalid MaxResources settings",
		},
		{
			name: "group maxresources exceed queue limits",
			config: QueueConfig{
				Name: "parent",
				Resources: Resources{
					Max: map[string]string{"memory": "100"},
				},
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 1,
						MaxResources:    map[string]string{"memory": "100"},
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group"},
						MaxApplications: 1,
						MaxResources:    map[string]string{"memory": "200"},
					},
				},
			},
			errMsg: "invalid MaxResources settings",
		},
		{
			name: "user maxapplications exceed queue limits",
			config: QueueConfig{
				Name:            "parent",
				MaxApplications: 100,
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 200,
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group"},
						MaxApplications: 100,
					},
				},
			},
			errMsg: "invalid MaxApplications settings",
		},
		{
			name: "group maxapplications exceed queue limits",
			config: QueueConfig{
				Name:            "parent",
				MaxApplications: 100,
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 100,
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group"},
						MaxApplications: 200,
					},
				},
			},
			errMsg: "invalid MaxApplications settings",
		},
		{
			name: "user maxapplications is 0",
			config: QueueConfig{
				Name: "parent",
				Resources: Resources{
					Max: map[string]string{"memory": "100"},
				},
				MaxApplications: 100,
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 0,
						MaxResources:    map[string]string{"memory": "100"},
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group"},
						MaxApplications: 100,
						MaxResources:    map[string]string{"memory": "100"},
					},
				},
			},
			errMsg: "",
		},
		{
			name: "group maxapplications is 0",
			config: QueueConfig{
				Name: "parent",
				Resources: Resources{
					Max: map[string]string{"memory": "100"},
				},
				MaxApplications: 100,
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user"},
						MaxApplications: 100,
						MaxResources:    map[string]string{"memory": "100"},
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group"},
						MaxApplications: 0,
						MaxResources:    map[string]string{"memory": "100"},
					},
				},
			},
			errMsg: "",
		},
		{
			name: "user wildcard is not last entry",
			config: QueueConfig{
				Name: "parent",
				Resources: Resources{
					Max: map[string]string{"memory": "100"},
				},
				MaxApplications: 100,
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"*", "test-user"},
						MaxResources:    map[string]string{"memory": "100"},
						MaxApplications: 100,
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"test-group", "*"},
						MaxResources:    map[string]string{"memory": "100"},
						MaxApplications: 100,
					},
				},
			},
			errMsg: "should not set no wildcard user test-user after wildcard user limit",
		},
		{
			name: "group wildcard is not last entry",
			config: QueueConfig{
				Name: "parent",
				Resources: Resources{
					Max: map[string]string{"memory": "100"},
				},
				MaxApplications: 100,
				Limits: []Limit{
					{
						Limit:           "user-limit",
						Users:           []string{"test-user", "*"},
						MaxResources:    map[string]string{"memory": "100"},
						MaxApplications: 100,
					},
					{
						Limit:           "group-limit",
						Groups:          []string{"*", "test-group"},
						MaxResources:    map[string]string{"memory": "100"},
						MaxApplications: 100,
					},
				},
			},
			errMsg: "should not set no wildcard group test-group after wildcard group limit",
		},
		{
			name: "group wildcard is only entry",
			config: QueueConfig{
				Name: "parent",
				Resources: Resources{
					Max: map[string]string{"memory": "100"},
				},
				MaxApplications: 100,
				Limits: []Limit{
					{
						Limit:           "group-limit",
						Groups:          []string{"*"},
						MaxResources:    map[string]string{"memory": "100"},
						MaxApplications: 100,
					},
				},
			},
			errMsg: "should not specify only one group limit that is using the wildcard.",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := checkLimits(testCase.config.Limits, testCase.config.Name, &testCase.config)
			if testCase.errMsg != "" {
				assert.ErrorContains(t, err, testCase.errMsg)
			} else {
				assert.NilError(t, err)
			}
		})
	}
}
