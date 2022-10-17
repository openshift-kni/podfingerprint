/*
 * Copyright 2022 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package podfingerprint

import (
	"encoding/json"
	"math/rand"
	"testing"
)

func TestNamespacedNameString(t *testing.T) {
	nn := NamespacedName{
		Namespace: "foo",
		Name:      "bar",
	}

	expected := "foo/bar"
	got := nn.String()
	if got != expected {
		t.Errorf("string failed: got %q expected %q", got, expected)
	}
}

func TestNamespacedNameGetters(t *testing.T) {
	nn := NamespacedName{
		Namespace: "foo",
		Name:      "bar",
	}

	nsGot := nn.GetNamespace()
	nGot := nn.GetName()
	if nsGot != "foo" || nGot != "bar" {
		t.Errorf("getters failed: %q vs %q and %q vs %q", nsGot, "foo", nGot, "bar")
	}
}

var expectedStatusJson string = `{"fingerprintExpected":"pfp0v001b92008c14168b3a6","fingerprintComputed":"pfp0v001b92008c14168b3a6","pods":[{"Namespace":"ns1","Name":"n1"},{"Namespace":"ns1","Name":"n2"},{"Namespace":"ns2","Name":"n1"},{"Namespace":"ns3","Name":"n1"},{"Namespace":"ns3","Name":"n2"}]}`

func TestTraceStatus(t *testing.T) {
	pods := []NamespacedName{
		{
			Namespace: "ns1",
			Name:      "n1",
		},
		{
			Namespace: "ns1",
			Name:      "n2",
		},
		{
			Namespace: "ns2",
			Name:      "n1",
		},
		{
			Namespace: "ns3",
			Name:      "n1",
		},
		{
			Namespace: "ns3",
			Name:      "n2",
		},
	}

	st := Status{}
	fp := NewTracingFingerprint(len(pods), &st)
	for _, pod := range pods {
		fp.Add(pod.Namespace, pod.Name)
	}
	fp.Sign()
	err := fp.Check("pfp0v001b92008c14168b3a6")
	if err != nil {
		t.Fatalf("fp check error: %v", err)
	}

	data, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}
	got := string(data)
	if got != expectedStatusJson {
		t.Errorf("status report error.\ngot: %s\nexp: %s", got, expectedStatusJson)
	}
}

func TestSignCrosscheck(t *testing.T) {
	if len(pods) == 0 || podsErr != nil {
		t.Fatalf("cannot load the test data: %v", podsErr)
	}

	localPods := make([]NamespacedName, len(pods))
	copy(localPods, pods)
	rand.Shuffle(len(localPods), func(i, j int) {
		localPods[i], localPods[j] = localPods[j], localPods[i]
	})

	fp := NewFingerprint(0)
	for _, pod := range pods {
		fp.Add(pod.Namespace, pod.Name)
	}
	fp2 := NewTracingFingerprint(0, NullTracer{})
	for _, localPod := range localPods {
		fp2.Add(localPod.Namespace, localPod.Name)
	}

	x := fp.Sign()
	x2 := fp2.Sign()
	if x != x2 {
		t.Fatalf("signature not stable: %q vs %q", x, x2)
	}
}
