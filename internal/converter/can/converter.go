// Copyright 2023 EMQ Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package can

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lf-edge/ekuiper/internal/conf"

	"github.com/ngjaying/can"
	"github.com/ngjaying/can/pkg/descriptor"
	"github.com/ngjaying/can/pkg/generate"

	"github.com/lf-edge/ekuiper/pkg/message"
)

// The converter for socketCan format
// Expect to receive a socketCan bytes array [16]byte with canId and data inside

type packedFrames struct {
	Meta   map[string]interface{} `json:"meta,omitempty"`
	Frames []can.Frame            `json:"frames,omitempty"`
}

type Converter struct {
	messages map[uint32]*descriptor.Message
}

func (c *Converter) Encode(_ interface{}) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (c *Converter) Decode(b []byte) (interface{}, error) {
	//frame := socketcan.Frame{}
	//frame.UnmarshalBinary(b)
	//if frame.IsError() {
	//	return nil, fmt.Errorf("error frame received: %v", frame.DecodeErrorFrame())
	//}
	//canFrame := frame.DecodeFrame()
	p := &packedFrames{}
	err := json.Unmarshal(b, p)
	//canFrame := &can.Frame{}
	//err := canFrame.UnmarshalJSON(b)
	if err != nil {
		return nil, fmt.Errorf("invalid frame json `%s` received: %v", b, err)
	}
	if p.Frames == nil {
		return nil, fmt.Errorf("invalid frame json `%s`, no frames", b)
	}
	result := make(map[string]interface{})
	for _, frame := range p.Frames {
		desc, ok := c.messages[frame.ID]
		if !ok {
			conf.Log.Errorf("cannot find message %d", frame.ID)
			continue
		}
		desc.DecodeToMap(&frame, result)
	}
	return result, nil
}

func NewConverter(dbcPath string) (message.Converter, error) {
	dir, err := os.Stat(dbcPath)
	if err != nil {
		return nil, err
	}
	mm := make(map[uint32]*descriptor.Message)
	if dir.IsDir() {
		var (
			files []string
			max   int64
		)
		err = filepath.Walk(dbcPath, func(path string, info os.FileInfo, err error) error {
			if strings.EqualFold(filepath.Ext(path), ".dbc") {
				if info.Size() > max {
					max = info.Size()
				}
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		b1 := make([]byte, max)
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				return nil, err
			}
			n, err := f.Read(b1)
			if err != nil {
				return nil, err
			}
			err = addMessageDb(mm, file, b1[:n])
			if err != nil {
				return nil, err
			}
		}
	} else {
		dbc, err := os.ReadFile(dbcPath)
		if nil != err {
			return nil, err
		}
		err = addMessageDb(mm, dbcPath, dbc)
		if err != nil {
			return nil, err
		}
	}
	return &Converter{
		messages: mm,
	}, nil
}

func addMessageDb(mm map[uint32]*descriptor.Message, dbcPath string, dbcContent []byte) error {
	c, err := generate.Compile(dbcPath, dbcContent)
	if err != nil {
		return err
	}
	for _, m := range c.Database.Messages {
		if _, ok := mm[m.ID]; !ok {
			mm[m.ID] = m
		}
	}
	return nil
}
