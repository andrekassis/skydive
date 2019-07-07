/*
 * Copyright (C) 2019 IBM, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy ofthe License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specificlanguage governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/skydive-project/skydive/config"
	"github.com/skydive-project/skydive/contrib/pipelines/core"
	"github.com/skydive-project/skydive/contrib/pipelines/secadvisor/sadns"
	"github.com/skydive-project/skydive/logging"
	"github.com/skydive-project/skydive/websocket"
)

// newTransformer returns a new transformer based on config file
func newTransformer(cfg *viper.Viper) (core.Transformer, error) {
	if cfg.GetBool(core.CfgRoot + "dns") {
		return sadns.NewTransformDNS()
	}
	return newTransform(cfg)
}

// newClassifier returns a new classifier based on config file
func newClassifier(cfg *viper.Viper) (core.Classifier, error) {
	if cfg.GetBool(core.CfgRoot + "dns") {
		return sadns.NewClassifyDNS()
	}
	return core.NewClassify(cfg)
}

// newFilter returns a new filter based on config file
func newFilter(cfg *viper.Viper) (core.Filterer, error) {
	if cfg.GetBool(core.CfgRoot + "dns") {
		return core.NewFilter("ingress", "egress", "internal", "other", "NODNS")
	}
	return core.NewFilterFromConfig(cfg)
}

// NewSubscriberFromConfig returns a new flow subscriber writing to object store
func NewSubscriberFromConfig(cfg *viper.Viper) (*websocket.StructSpeaker, error) {
	transform, err := newTransformer(cfg)
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize transform: %s", err)
	}

	classify, err := newClassifier(cfg)
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize classify: %s", err)
	}

	filter, err := newFilter(cfg)
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize filter: %s", err)
	}

	encode, err := core.NewEncodeJSON()
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize encode: %s", err)
	}

	compress, err := core.NewCompressGzip()
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize compress: %s", err)
	}

	store, err := core.NewStoreFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize store: %s", err)
	}

	pipeline := core.NewPipeline(transform, classify, filter, encode, compress, store)

	return core.NewSubscriber(pipeline, cfg)
}

func main() {
	defaultCfgFile := "/etc/skydive/secadvisor.yml"
	if len(os.Args) > 1 {
		defaultCfgFile = os.Args[1]
	}

	if err := config.InitConfig("file", []string{defaultCfgFile}); err != nil {
		logging.GetLogger().Errorf("Failed to initialize config: %s", err)
		os.Exit(1)
	}

	if err := config.InitLogging(); err != nil {
		logging.GetLogger().Errorf("Failed to initialize logging system: %s", err)
		os.Exit(1)
	}

	s, err := NewSubscriberFromConfig(config.GetConfig())
	if err != nil {
		logging.GetLogger().Errorf("Failed to initialize subscriber: %s", err)
		os.Exit(1)
	}

	core.SubscriberRun(s)
}