/*
 * Copyright 2023 Red Hat, Inc.
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

package pfpstatus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/k8stopologyawareschedwg/podfingerprint"
)

const (
	BaseDirectory = "/run/pfpstatus"
)

var (
	pfpStatusDir string = BaseDirectory // mostly test aid
	pfpUpdatesCh chan *StatusInfo
)

type TracingStatus struct {
	nodeName string
	data     podfingerprint.Status
	updates  chan *StatusInfo
}

func MakeTracingStatus(nodeName string, updates chan *StatusInfo) TracingStatus {
	if updates == nil {
		updates = pfpUpdatesCh
	}
	return TracingStatus{
		nodeName: nodeName,
		updates:  updates,
		data:     podfingerprint.MakeStatus(nodeName),
	}
}

func (st *TracingStatus) Start(numPods int) {
	st.data.Start(numPods)
}

func (st *TracingStatus) Add(namespace, name string) {
	st.data.Add(namespace, name)
}

func (st *TracingStatus) Sign(computed string) {
	st.data.Sign(computed)
}

func (st *TracingStatus) Check(expected string) {
	st.data.Check(expected)

	if st.updates == nil || pfpStatusDir == "" {
		return
	}
	info := StatusInfo{
		NodeName:     st.nodeName,
		Data:         st.data.Clone(),
		LastModified: time.Now(),
	}
	st.updates <- &info
}

func (st TracingStatus) Repr() string {
	return st.data.Repr()
}

type StatusInfo struct {
	NodeName     string                `json:"nodeName"`
	Data         podfingerprint.Status `json:"data"`
	LastModified time.Time             `json:"lastModified"`
}

func DumpNodeStatus(statusDir string, st *StatusInfo) error {
	dst, err := os.CreateTemp(statusDir, st.NodeName)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(dst).Encode(st); err != nil {
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Rename(dst.Name(), filepath.Join(statusDir, st.NodeName+".json"))
}

func LoadNodeStatus(statusDir, nodeName string) (*StatusInfo, error) {
	src, err := os.Open(filepath.Join(statusDir, nodeName+".json"))
	defer src.Close()
	if err != nil {
		return nil, err
	}
	var st StatusInfo
	err = json.NewDecoder(src).Decode(&st)
	return &st, err
}

func RunForever(ctx context.Context, logger logr.Logger, baseDirectory string, updates chan *StatusInfo) {
	pfpStatusDir = baseDirectory
	pfpUpdatesCh = updates

	// let's try to keep the amount of code we do in init() at minimum.
	// This may happen if the container didn't have the directory mounted
	discard := !existsBaseDirectory(pfpStatusDir)
	if discard {
		logger.Info("base directory not found, will discard everything", "baseDirectory", pfpStatusDir)
	}

	logger.V(4).Info("status update loop started")
	defer logger.V(4).Info("status update loop finished")
	for {
		select {
		case <-ctx.Done():
			return
		case st := <-updates:
			if discard {
				return
			}
			// intentionally ignore errors, must keep going
			DumpNodeStatus(BaseDirectory, st)
		}
	}
}

func existsBaseDirectory(baseDir string) bool {
	info, err := os.Stat(baseDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}
